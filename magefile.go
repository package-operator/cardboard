//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/mt-sre/devkube/magedeps"
)

// Directories
var (
	// Working directory of the project.
	workDir string
	// Cache directory for temporary build files.
	cacheDir string
	// Dependency directory.
	depsDir magedeps.DependencyDirectory
)

func init() {
	var err error
	// Directories
	workDir, err = os.Getwd()
	if err != nil {
		panic(fmt.Errorf("getting work dir: %w", err))
	}

	depsDir = magedeps.DependencyDirectory(filepath.Join(workDir, ".deps"))
	cacheDir = filepath.Join(workDir, ".cache")

	// Path
	os.Setenv("PATH", depsDir.Bin()+":"+os.Getenv("PATH"))
}

// Runs go mod tidy in all go modules of the repository.
func Tidy() error {
	if err := sh.Run("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("tidy main module: %w", err)
	}
	return nil
}

// Testing and Linting
// -------------------

type Test mg.Namespace

// Runs unittests.
func (Test) Unit() error {
	return sh.RunWithV(
		map[string]string{"CGO_ENABLED": "1"},
		"go", "test", "-v",
		fmt.Sprintf("-coverprofile=%s", filepath.Join(cacheDir, "cov.out")), "-race",
		"./dev/...", "./cmd/...", "./magedeps/...",
	)
}

// Lints the source code.
func (Test) Lint() error {
	mg.Deps(Dependency.GolangciLint)

	for _, cmd := range [][]string{
		{"go", "fmt", "./..."},
		{"golangci-lint", "run", "./...", "--deadline=15m"},
	} {
		if err := sh.RunV(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("running %q: %w", strings.Join(cmd, " "), err)
		}
	}
	return nil
}

// Runs integration tests
func (Test) Integration() error {
	return sh.RunWithV(map[string]string{
		"CGO_ENABLED": "1",
	}, "go", "test", "-cover", "-v", "-race", "./integration/...")
}

// Dependencies
// ------------

// Versions
const (
	goimportsVersion    = "0.11.1"
	golangciLintVersion = "1.53.3"
)

type Dependency mg.Namespace

func (Dependency) Goimports() error {
	return depsDir.GoInstall("go-imports",
		"golang.org/x/tools/cmd/goimports", goimportsVersion)
}

func (Dependency) GolangciLint() error {
	return depsDir.GoInstall("golangci-lint",
		"github.com/golangci/golangci-lint/cmd/golangci-lint", golangciLintVersion)
}
