module pkg.package-operator.run/cardboard/modules/oci

go 1.22.0

toolchain go1.22.2

require (
	pkg.package-operator.run/cardboard v0.0.3
	pkg.package-operator.run/cardboard/kubeutils v0.0.3
)

replace (
	pkg.package-operator.run/cardboard => ../../
	pkg.package-operator.run/cardboard/kubeutils => ../../kubeutils
)
