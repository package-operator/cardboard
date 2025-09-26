module pkg.package-operator.run/cardboard/modules/oci

go 1.24.6

require (
	pkg.package-operator.run/cardboard v0.0.4
	pkg.package-operator.run/cardboard/kubeutils v0.0.4
)

replace (
	pkg.package-operator.run/cardboard => ../../
	pkg.package-operator.run/cardboard/kubeutils => ../../kubeutils
)
