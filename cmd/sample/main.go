package main

import (
	"context"
	"embed"
	"fmt"
	"os"

	"pkg.package-operator.run/cardboard/run"
)

var (
	mgr *run.Manager

	//go:embed *.go
	source embed.FS
)

func main() {
	ctx := context.Background()

	mgr = run.New(run.WithSources(source))
	mgr.Register()
	if err := mgr.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s\n", err)
		os.Exit(1)
	}
}

func FuncTarget(ctx context.Context) error {
	self := run.Fn(FuncTarget)
	return mgr.ParallelDeps(ctx, self, run.Fn(func() {
		fmt.Println("ok")
	}))
}

type (
	empty   struct{}
	suspect struct{}
)

func (s *suspect) Meth(ctx context.Context) error { return nil }

type coll struct{}

func (c *coll) MethTarget(ctx context.Context, args []string) error {
	self := run.Meth1(c, c.MethTarget, args)
	return mgr.ParallelDeps(ctx, self, run.Fn(FuncTarget))
}

func (c *coll) MethTargetWrongName(ctx context.Context, args []string) error {
	this := run.Meth1(c, c.MethTarget, args)
	return mgr.ParallelDeps(ctx, this, run.Fn(FuncTarget))
}

func (c *coll) MethTargetWrongInline(ctx context.Context, args []string) error {
	return mgr.ParallelDeps(ctx, run.Meth1(c, c.MethTarget, args), run.Fn(FuncTarget))
}

func (c *coll) MethTargetWrongInlineAndBare(ctx context.Context, args []string) error {
	mgr.ParallelDeps(ctx, run.Meth1(c, c.MethTarget, args), run.Fn(FuncTarget))
	return nil
}

func (c *coll) MethTargetWrongInlineAndIf(ctx context.Context, args []string) error {
	if mgr.ParallelDeps(ctx, run.Meth1(c, c.MethTarget, args), run.Fn(FuncTarget)) != nil {
		return nil
	}
	return nil
}

func (c *coll) MethTargetWrongInlineAndIfBlock(ctx context.Context, args []string) error {
	if err := mgr.ParallelDeps(ctx, run.Meth1(c, c.MethTarget, args), run.Fn(FuncTarget)); err != nil {
		return err
	}
	return nil
}

func (c *coll) MethNoArgs(ctx context.Context) error {
	if err := mgr.ParallelDeps(ctx, run.Meth(c, c.MethNoArgs), run.Fn(FuncTarget)); err != nil {
		return err
	}
	return nil
}

func (c *coll) MethTwoArgs(ctx context.Context, foo, bar string) error {
	if err := mgr.ParallelDeps(ctx, run.Meth2(c, c.MethTwoArgs, foo, bar), run.Fn(FuncTarget)); err != nil {
		return err
	}
	return nil
}

func (c *coll) MethInvalidTwoArgs(ctx context.Context, foo, bar string) error {
	if err := mgr.ParallelDeps(ctx, run.Meth2(empty{}, c.MethTwoArgs, foo, bar), run.Fn(FuncTarget)); err != nil {
		return err
	}
	return nil
}

func (c *coll) MethInvalidWrongReceiverWithSameSignatureReceiverTwoArgs(ctx context.Context, foo, bar string) error {
	if err := mgr.ParallelDeps(ctx, run.Meth2(suspect{}, c.MethTwoArgs, foo, bar), run.Fn(FuncTarget)); err != nil {
		return err
	}
	return nil
}

func (c *coll) MethExtractedTwoArgs(ctx context.Context, foo, bar string) error {
	methTwoArgs := c.MethTwoArgs
	if err := mgr.ParallelDeps(ctx, run.Meth2(c, methTwoArgs, foo, bar), run.Fn(FuncTarget)); err != nil {
		return err
	}
	return nil
}

func (c *coll) MethInvalidExtracted(ctx context.Context) error {
	meth := (&suspect{}).Meth
	if err := mgr.ParallelDeps(ctx, run.Meth(c, meth)); err != nil {
		return err
	}
	return nil
}
