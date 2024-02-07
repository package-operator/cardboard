module pkg.package-operator.run/cardboard/modules/oci

go 1.21

require (
	pkg.package-operator.run/cardboard v0.0.0-20240207155344-4e27bb34423a
	pkg.package-operator.run/cardboard/kubeutils v0.0.0-20240207155344-4e27bb34423a
)

replace (
	pkg.package-operator.run/cardboard => ../../
	pkg.package-operator.run/cardboard/kubeutils => ../../kubeutils
)
