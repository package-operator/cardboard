// Package main contains the build files for cardboard.
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"

	"pkg.package-operator.run/cardboard/run"
	"pkg.package-operator.run/cardboard/sh"
)

var (
	shr *sh.Runner
	mgr *run.Manager

	// internal modules.
	test *Test
	lint *Lint

	//go:embed *.go
	source embed.FS
)

func main() {
	ctx := context.Background()

	mgr = run.New(run.WithSources(source))
	shr = sh.New()

	test = &Test{}
	lint = &Lint{}

	err := errors.Join(
		mgr.RegisterGoTool("gotestfmt", "github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt", "2.5.0"),
		mgr.RegisterGoTool("golangci-lint", "github.com/golangci/golangci-lint/cmd/golangci-lint", "1.60.1"),
		mgr.Register(&Dev{}, &CI{}),
	)
	if err != nil {
		panic(err)
	}

	if err := mgr.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

//
// Callable targets defined below.
//

// CI targets that should only be called within the CI/CD runners.
type CI struct{}

// Runs unittests in CI.
func (ci *CI) Unit(ctx context.Context, args []string) error {
	return commonUnit(ctx, args)
}

// Runs linters in CI to check the codebase.
func (ci *CI) Lint(_ context.Context, _ []string) error {
	return lint.Check()
}

func (ci *CI) PostPush(ctx context.Context, args []string) error {
	self := run.Meth1(ci, ci.PostPush, args)
	err := mgr.ParallelDeps(ctx, self,
		run.Meth(lint, lint.glciFix),
		run.Meth(lint, lint.goModTidyAll),
	)
	if err != nil {
		return err
	}

	return shr.Run("git", "diff", "--quiet", "--exit-code")
}

// Development focused commands using local development environment.
type Dev struct{}

// Runs local unittests.
func (d *Dev) Unit(ctx context.Context, args []string) error {
	return commonUnit(ctx, args)
}

// Runs local linters to check the codebase.
func (d *Dev) Lint(_ context.Context, _ []string) error {
	return lint.Check()
}

// Runs linters and code-gens for pre-commit.
func (d *Dev) PreCommit(ctx context.Context, args []string) error {
	self := run.Meth1(d, d.PreCommit, args)
	return mgr.ParallelDeps(ctx, self,
		run.Meth(lint, lint.glciFix),
		run.Meth(lint, lint.goModTidyAll),
	)
}

// Tries to fix linter issues.
func (d *Dev) LintFix(_ context.Context, _ []string) error {
	return lint.Fix()
}

// common unittest target shared by CI and Dev.
func commonUnit(ctx context.Context, args []string) error {
	var filter string
	switch len(args) {
	case 0:
		// nothing
	case 1:
		filter = args[0]
	default:
		return errors.New("only supports a single argument") //nolint:goerr113
	}
	return test.Unit(ctx, filter)
}
