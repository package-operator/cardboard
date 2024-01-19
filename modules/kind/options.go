package kind

import (
	"pkg.package-operator.run/cardboard/kubeutils"
	kindv1alpha4 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
)

type WithContainerRuntime kubeutils.ContainerRuntime

func (cr WithContainerRuntime) ApplyToCluster(kc *Cluster) {
	kc.containerRuntime = kubeutils.ContainerRuntime(cr)
}

type WithClusterConfig kindv1alpha4.Cluster

func (opts WithClusterConfig) ApplyToCluster(kc *Cluster) {
	config := kindv1alpha4.Cluster(opts)
	kc.clusterConfig = &config
}
