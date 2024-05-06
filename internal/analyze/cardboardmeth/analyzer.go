package cardboardmeth

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"reflect"
	"regexp"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const (
	cardboardRunPackage = "pkg.package-operator.run/cardboard/run"
)

var methFnRegex = regexp.MustCompile("^Meth(.*)")

var Analyzer = &analysis.Analyzer{
	Name: "cardboardmeth",
	Doc:  "Checks that cardboard run.MethX calls pass correct struct and method pairs.",
	Run:  run,

	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		ce := n.(*ast.CallExpr)

		fn := typeutil.Callee(pass.TypesInfo, ce)
		if fn == nil {
			return
		}
		if pkg := fn.Pkg(); pkg == nil || pkg.Path() != cardboardRunPackage {
			return // This analyzer is only interested in calls to types from the `cardboardRunPackage` package.
		}

		recv := fn.Type().(*types.Signature).Recv()
		if recv != nil {
			return // This analyzer is only interested in function calls.
		}

		captures := methFnRegex.FindStringSubmatch(fn.Name())
		if captures == nil {
			return // This analyzer is only interested in calls to run.MethX.
		}

		if raw := captures[1]; raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				pass.Reportf(ce.Pos(), "call ro run.MethX with unparseable arity: %s %q", err, render(pass.Fset, ce))
				return
			}
			if parsed < 1 {
				pass.Reportf(ce.Pos(), "call ro run.MethX with arity <1: %q", render(pass.Fset, ce))
				return
			}
		}

		groupType := pass.TypesInfo.TypeOf(ce.Args[0])

		var xType types.Type
		switch meth := ce.Args[1].(type) {
		case *ast.SelectorExpr:
			xType = pass.TypesInfo.TypeOf(meth.X)
			if !reflect.DeepEqual(xType, groupType) {
				pass.Reportf(ce.Args[1].Pos(), "method %q does not belong to type of %q: %q %s", render(pass.Fset, ce.Args[1]), render(pass.Fset, ce.Args[0]), render(pass.Fset, ce), xType)
			}
		case *ast.Ident:
			pass.Reportf(ce.Args[1].Pos(), "identifier %q should be removed and method selector expression inlined: %q", render(pass.Fset, ce.Args[1]), render(pass.Fset, ce))
		default:
			pass.Reportf(ce.Args[1].Pos(), "not implemented: %q has method arg node type %T", render(pass.Fset, ce), meth)
		}
	})

	return nil, nil
}

// from https://arslan.io/2019/06/13/using-go-analysis-to-write-a-custom-linter/
func render(fset *token.FileSet, x interface{}) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, x); err != nil {
		panic(err)
	}
	return buf.String()
}
