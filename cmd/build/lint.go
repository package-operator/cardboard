package main

import (
	"context"

	"pkg.package-operator.run/cardboard/run"
	"pkg.package-operator.run/cardboard/sh"
)

// internal struct to namespace all lint related functions.
type Lint struct{}

func (l Lint) Fix() error   { return l.glciFix() }
func (l Lint) Check() error { return l.glciCheck() }

// TODO make that more nice.
func (l Lint) goModTidy(workdir string) error {
	return shr.New(sh.WithWorkDir(workdir)).Run("go", "mod", "tidy")
}

func (l Lint) goModTidyAll(ctx context.Context) error {
	return mgr.ParallelDeps(ctx, run.Meth(l, l.goModTidyAll),
		run.Meth1(l, l.goModTidy, "."),
		run.Meth1(l, l.goModTidy, "./kubeutils/"),
		run.Meth1(l, l.goModTidy, "./modules/kind/"),
		run.Meth1(l, l.goModTidy, "./modules/kubeclients/"),
		run.Meth1(l, l.goModTidy, "./modules/oci/"),
	)
}

func (Lint) glciFix() error {
	return shr.Run("golangci-lint", "run", "--fix",
		"./...", "./kubeutils/...", "./modules/kind/...", "./modules/kubeclients/...", "./modules/oci/...",
	)
}

func (Lint) glciCheck() error {
	return shr.Run("golangci-lint", "run",
		"./...", "./kubeutils/...", "./modules/kind/...", "./modules/kubeclients/...", "./modules/oci/...",
	)
}
