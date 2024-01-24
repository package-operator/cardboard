module pkg.package-operator.run/cardboard/modules/oci

go 1.21

require (
	pkg.package-operator.run/cardboard v0.0.0-00010101000000-000000000000
	pkg.package-operator.run/cardboard/kubeutils v0.0.0-00010101000000-000000000000
)

replace (
	pkg.package-operator.run/cardboard => ../../
	pkg.package-operator.run/cardboard/kubeutils => ../../kubeutils
)
