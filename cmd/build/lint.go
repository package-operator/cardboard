package main

import (
	"context"

	"pkg.package-operator.run/cardboard/run"
	"pkg.package-operator.run/cardboard/sh"
)

// internal struct to namespace all lint related functions.
type Lint struct{}

func (l Lint) Fix(ctx context.Context) error   { return l.glciFix(ctx) }
func (l Lint) Check(ctx context.Context) error { return l.glciCheck(ctx) }

// TODO make that more nice.
func (l Lint) goModTidy(ctx context.Context, workdir string) error {
	return shr.New(sh.WithWorkDir(workdir)).Run(ctx, "go", "mod", "tidy")
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

func (Lint) glciFix(ctx context.Context) error {
	return shr.Run(ctx, "golangci-lint", "run", "--timeout", "2m", "--fix",
		"./...", "./kubeutils/...", "./modules/kind/...", "./modules/kubeclients/...", "./modules/oci/...",
	)
}

func (Lint) glciCheck(ctx context.Context) error {
	return shr.Run(ctx, "golangci-lint", "run", "--timeout", "2m",
		"./...", "./kubeutils/...", "./modules/kind/...", "./modules/kubeclients/...", "./modules/oci/...",
	)
}

func (Lint) goWorkSync(ctx context.Context) error {
	return shr.Run(ctx, "go", "work", "sync")
}
