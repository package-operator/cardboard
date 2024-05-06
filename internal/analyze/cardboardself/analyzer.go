package cardboardself

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name: "cardboardself",
	Doc:  "Checks that self is a wrapped version of the caller.",
	Run:  run,

	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{}

	inspect.WithStack(nodeFilter, func(n ast.Node, _ bool, stack []ast.Node) bool {
		return false
	})

	return nil, nil
}
