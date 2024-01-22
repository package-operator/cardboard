package kind

import (
	"context"

	"pkg.package-operator.run/cardboard/kubeutils"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

type WithClusterInitializers []ClusterInitializer

func (opts WithClusterInitializers) ApplyToCluster(kc *Cluster) {
	kc.initializers = []ClusterInitializer(opts)
}

type ClusterInitializer interface {
	Init(ctx context.Context, cluster *Cluster) error
}

// Run a function with access to the cluster object. Can be used to directly interact with the cluster.
type ClusterInitFn func(ctx context.Context, cluster *Cluster) error

func (fn ClusterInitFn) Init(ctx context.Context, cl *Cluster) error {
	return fn(ctx, cl)
}

// Load objects from given folder paths and applies them into the cluster.
type ClusterLoadObjectsFromFolders []string

func (l ClusterLoadObjectsFromFolders) Init(
	ctx context.Context, cluster *Cluster) error {
	return cluster.clients.CreateAndWaitFromFolders(ctx, l)
}

// Load objects from given file paths and applies them into the cluster.
type ClusterLoadObjectsFromFiles []string

func (l ClusterLoadObjectsFromFiles) Init(
	ctx context.Context, cluster *Cluster) error {
	return cluster.clients.CreateAndWaitFromFiles(ctx, l)
}

// Load objects from the given http urls and applies them into the cluster.
type ClusterLoadObjectsFromHttp []string

func (l ClusterLoadObjectsFromHttp) Init(
	ctx context.Context, cluster *Cluster) error {
	return cluster.clients.CreateAndWaitFromHTTP(ctx, l)
}

// Creates the referenced Object and waits for it to be ready.
type ClusterLoadObjectFromClientObject struct {
	client.Object
}

func (l ClusterLoadObjectFromClientObject) Init(
	ctx context.Context, cluster *Cluster) error {
	return cluster.clients.CreateAndWaitForReadiness(ctx, l.Object)
}
