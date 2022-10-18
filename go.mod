module github.com/mt-sre/devkube

go 1.16

require (
	github.com/go-logr/logr v1.2.0
	github.com/magefile/mage v1.12.1
	github.com/openshift/addon-operator/apis v0.0.0-20220111092509-93ca25c9359f
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.23.0
	k8s.io/apiextensions-apiserver v0.23.0
	k8s.io/apimachinery v0.23.1
	k8s.io/client-go v0.23.0
	sigs.k8s.io/controller-runtime v0.11.0
	sigs.k8s.io/yaml v1.3.0
)
