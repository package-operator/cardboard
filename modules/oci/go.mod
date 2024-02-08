module pkg.package-operator.run/cardboard/modules/oci

go 1.21

require (
	pkg.package-operator.run/cardboard v0.0.0-20240208102837-3d27746a51bc
	pkg.package-operator.run/cardboard/kubeutils v0.0.0-20240208102837-3d27746a51bc
)

replace (
	pkg.package-operator.run/cardboard => ../../
	pkg.package-operator.run/cardboard/kubeutils => ../../kubeutils
)
