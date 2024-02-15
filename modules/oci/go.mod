module pkg.package-operator.run/cardboard/modules/oci

go 1.21

require (
	pkg.package-operator.run/cardboard v0.0.0-20240215101355-db99fcc2d2ce
	pkg.package-operator.run/cardboard/kubeutils v0.0.0-20240215101355-db99fcc2d2ce
)

replace (
	pkg.package-operator.run/cardboard => ../../
	pkg.package-operator.run/cardboard/kubeutils => ../../kubeutils
)
