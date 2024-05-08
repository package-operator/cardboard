package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"pkg.package-operator.run/cardboard/internal/analyze/cardboardmeth"
	"pkg.package-operator.run/cardboard/internal/analyze/cardboardselfident"
)

func main() {
	multichecker.Main(
		cardboardselfident.Analyzer,
		cardboardmeth.Analyzer,
	)
}
