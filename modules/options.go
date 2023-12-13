package modules

import (
	kindv1alpha4 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
)

type WithContainerRuntime ContainerRuntime

func (cr WithContainerRuntime) ApplyToKindCluster(kc *KindCluster) {
	kc.containerRuntime = ContainerRuntime(cr)
}

func (cr WithContainerRuntime) ApplyToOCI(oci *OCI) {
	oci.containerRuntime = ContainerRuntime(cr)
}

type WithKindClusterConfig kindv1alpha4.Cluster

func (opts WithKindClusterConfig) ApplyToKindCluster(kc *KindCluster) {
	config := kindv1alpha4.Cluster(opts)
	kc.kindClusterConfig = &config
}
