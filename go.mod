module pkg.package-operator.run/cardboard

go 1.23.0

toolchain go1.23.4

replace (
	pkg.package-operator.run/cardboard/kubeutils => ./kubeutils
	pkg.package-operator.run/cardboard/modules/kind => ./modules/kind
	pkg.package-operator.run/cardboard/modules/kubeclients => ./modules/kubeclients
	pkg.package-operator.run/cardboard/modules/oci => ./modules/oci
)

require (
	github.com/mattn/go-isatty v0.0.20
	github.com/neilotoole/slogt v1.1.0
	github.com/stretchr/testify v1.10.0
	github.com/xlab/treeprint v1.2.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	golang.org/x/sys v0.31.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
