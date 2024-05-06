package cardboardselfident

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"reflect"
	"regexp"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const (
	cardboardRunPackage = "pkg.package-operator.run/cardboard/run"
	runManagerTypeName  = "*pkg.package-operator.run/cardboard/run.Manager"
)

var depsFnRegex = regexp.MustCompile("^(Serial|Parallel)Deps$")

var Analyzer = &analysis.Analyzer{
	Name: "cardboardselfident",
	Doc:  "Checks that cardboard targets pass correct self arg to their dependencies.",
	Run:  run,

	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

// playground to explore a linter that enforces a separate non-inline `self` argument in calls to mgr.Serial/ParallelDeps

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	inspect.WithStack(nodeFilter, func(n ast.Node, _ bool, stack []ast.Node) bool {
		ce := n.(*ast.CallExpr)

		fn := typeutil.Callee(pass.TypesInfo, ce)
		if fn == nil {
			return false
		}
		if pkg := fn.Pkg(); pkg == nil || pkg.Path() != cardboardRunPackage {
			return false // This analyzer is only interested in calls to types from the `cardboardRunPackage` package.
		}

		recv := fn.Type().(*types.Signature).Recv()
		if recv == nil {
			return false // This analyzer is only interested in method calls.
		}
		recvName := recv.Type().Underlying().String()
		if recvName != runManagerTypeName {
			return false // This analyzer is only interested in calls to the `runManagerTypeName` type.
		}

		if !depsFnRegex.MatchString(fn.Name()) {
			return false // This analyzer is only interested in calls to Parallel-/SerialDeps.
		}

		parentIndex, parentBlock := nextBlock(stack)

		// second arg should be `self`
		if selfIdent, ok := ce.Args[1].(*ast.Ident); !ok {
			// second arg is not an identifier
			invalidSelfArgExpr := ce.Args[1]
			offendingCallSrc := render(pass.Fset, ce.Args[1])

			childStmt := stack[parentIndex+1]
			childIndex := -1
			for i, stmt := range parentBlock.List {
				if reflect.DeepEqual(stmt, childStmt) {
					childIndex = i
					break
				}
			}

			// prepare assignment statement `self := invalidSelfArgExpr`
			// https://yuroyoro.github.io/goast-viewer/index.html
			// turned out very helpful when I constructed the assignment statement below.
			assignment := &ast.AssignStmt{
				Tok: token.DEFINE,
				Lhs: []ast.Expr{ast.NewIdent("self")},
				Rhs: []ast.Expr{invalidSelfArgExpr},
			}

			// Inject the assignment just before the statement that contains the invalid call.
			list := make([]ast.Stmt, len(parentBlock.List)+1)
			copy(list, parentBlock.List[:childIndex])
			list[childIndex] = assignment
			copy(list[childIndex+1:], parentBlock.List[childIndex:])

			// Replace call arg with `self` identifier.
			ce.Args[1] = ast.NewIdent("self")
			// Replace parent block statement list.
			parentBlock.List = list

			pass.Report(analysis.Diagnostic{
				Pos:     ce.Pos(),
				Message: fmt.Sprintf(`Second arg to %s call should be identifier "self" but is expression %q.`, fn.Name(), offendingCallSrc),
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: fmt.Sprintf(`Separate %q out into separate short variable declaration.`, offendingCallSrc),
						TextEdits: []analysis.TextEdit{
							{
								Pos:     parentBlock.Pos(),
								End:     parentBlock.End(),
								NewText: []byte(render(pass.Fset, parentBlock)),
							},
						},
					},
				},
			})
		} else if selfIdent.Name != "self" {
			// second arg is an identifier but not named "self"
			pass.Reportf(ce.Pos(), `second arg to %s call should be identifier "self" but is "%s".`, fn.Name(), selfIdent.Name)
		}
		return false
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

// Finds surrounding block statement by travelling up the stack.
func nextBlock(stack []ast.Node) (int, *ast.BlockStmt) {
	for i := len(stack) - 1; i >= 0; i-- {
		block, ok := stack[i].(*ast.BlockStmt)
		if !ok {
			continue
		}
		return i, block
	}
	return -1, nil
}
