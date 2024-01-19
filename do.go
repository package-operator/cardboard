// BANANA!
package main

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"pkg.package-operator.run/cardboard/modules/kind"
	"pkg.package-operator.run/cardboard/run"
	"pkg.package-operator.run/cardboard/sh"
)

var (
	cluster = kind.NewCluster("banana")
	mgr     *run.Manager
)

//go:embed *.go
var source embed.FS

func main() {
	ctx := context.Background()
	mgr = run.New(run.WithSources(source))
	run.Must(mgr.RegisterGoTool("gotestfmt", "github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt", "2.5.0"))

	mgr.MustRegisterAndRun(ctx, &Test{})
}

type Test struct{}

func (t *Test) testErrDeepDep() error {
	return fmt.Errorf("banana err")
}

func (t *Test) testDeepDep() error {
	return nil
}

func (t *Test) testDep(ctx context.Context) {
	run.Must(mgr.SerialDeps(ctx,
		run.Meth(t, t.testDep),
		run.Meth(t, t.testDeepDep),
		run.Meth(t, t.testErrDeepDep),
	))
}

// Do stuff.
func (t *Test) Test(ctx context.Context, args []string) error {
	run.Must(mgr.ParallelDeps(
		ctx, run.Meth1(t, t.Test, args),
		run.Meth(t, t.testDep),
		run.Meth(t, t.testDeepDep),
	))

	return nil
}

// Create and setup the test cluster.
func (t *Test) Cluster(ctx context.Context, args []string) error {
	run.Must(mgr.SerialDeps(ctx, run.Meth1(t, t.Cluster, args), run.Fn(cluster.Create)))
	return nil
}

// Run unittests, first argument is passed as -run="" filter.
func (t *Test) Unit(_ context.Context, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("test:unit only supports a single argument")
	}
	gotestArgs := []string{
		"-coverprofile=cover.txt", "-race", "-json",
	}
	if len(args) == 1 {
		gotestArgs = append(gotestArgs, "-run="+args[0])
	}

	return sh.New(sh.WithEnvironment{"CGO_ENABLED": "1"}).Bash(
		"set -euo pipefail",
		"go test "+strings.Join(gotestArgs, " ")+" ./... 2>&1 | tee gotest.log | gotestfmt --hide=empty-packages",
	)
}
