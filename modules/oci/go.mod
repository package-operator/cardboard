module pkg.package-operator.run/cardboard/modules/oci

go 1.21.1

require (
	pkg.package-operator.run/cardboard v0.0.0-00010101000000-000000000000
	pkg.package-operator.run/cardboard/kubeutils v0.0.0-00010101000000-000000000000
)

replace (
	pkg.package-operator.run/cardboard => ../../
	pkg.package-operator.run/cardboard/kubeutils => ../../kubeutils
)

require (
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
)
