package main

import (
	"context"
	"fmt"
	"strings"

	"pkg.package-operator.run/cardboard/sh"
)

// internal struct to namespace all test related functions.
type Test struct{}

// Run unittests, the filter argument is passed via -run="".
func (t Test) Unit(_ context.Context, filter string) error {
	gotestArgs := []string{"-coverprofile=cover.txt", "-race", "-json"}
	if len(filter) > 0 {
		gotestArgs = append(gotestArgs, "-run="+filter)
	}

	argStr := strings.Join(gotestArgs, " ")

	return sh.New(
		sh.WithEnvironment{"CGO_ENABLED": "1"},
	).Bash(
		"set -euo pipefail",
		fmt.Sprintf("go test %s ./... 2>&1 | tee gotest.log | gotestfmt --hide=empty-packages", argStr),
	)
}
