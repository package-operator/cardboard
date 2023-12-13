package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	kindv1alpha4 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	kindcmd "sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/yaml"

	"pkg.package-operator.run/cardboard/run"
)

type KindCluster struct {
	name, workDir     string
	kubeconfigPath    string
	containerRuntime  ContainerRuntime
	kindClusterConfig *kindv1alpha4.Cluster

	clients  *KubeClients
	provider *cluster.Provider
}

type KindClusterOption interface {
	ApplyToKindCluster(kc *KindCluster)
}

func NewKindCluster(name string, opts ...KindClusterOption) *KindCluster {
	kc := &KindCluster{
		name:    name,
		workDir: filepath.Join(defaultCacheDirectory, "clusters", name),
	}
	kc.kubeconfigPath = filepath.Join(kc.workDir, "kubeconfig.yaml")
	for _, opt := range opts {
		opt.ApplyToKindCluster(kc)
	}
	if kc.kindClusterConfig == nil {
		kc.kindClusterConfig = &kindv1alpha4.Cluster{}
	}
	sanitizeKindClusterConfig(kc.kindClusterConfig)
	defaultKindClusterConfig(kc.kindClusterConfig)
	return kc
}

func (c *KindCluster) KubeconfigPath() string {
	return c.kubeconfigPath
}

func (c *KindCluster) ID() string {
	return fmt.Sprintf("pkg.package-operator.run/cardboard/modules.KindCluster{name:%s}", c.name)
}

// Check if the cluster already exists.
func (c *KindCluster) Exists() (bool, error) {
	provider, err := c.getKindProvider()
	if err != nil {
		return false, err
	}
	existingKindClusters, err := provider.List()
	if err != nil {
		return false, fmt.Errorf("failed to fetch the existing KinD clusters: %w", err)
	}
	for _, cluster := range existingKindClusters {
		if c.name == cluster {
			return true, nil
		}
	}
	return false, nil
}

func (c *KindCluster) Clients() (*KubeClients, error) {
	if c.clients != nil {
		return c.clients, nil
	}
	c.clients = NewKubeClients(c.kubeconfigPath)
	if err := c.clients.Run(context.Background()); err != nil {
		return nil, err
	}
	return c.clients, nil
}

// Returns a Create dependency for stetting up pre-requisites.
func (c *KindCluster) CreateDep() run.Dependency {
	return run.Meth(c, c.Create)
}

// Returns a Destroy dependency for stetting up pre-requisites.
func (c *KindCluster) DestroyDep() run.Dependency {
	return run.Meth(c, c.Destroy)
}

// Creates the KinD cluster if it does not exist.
func (c *KindCluster) Create() error {
	var err error
	c.containerRuntime, err = c.containerRuntime.Get()
	if err != nil {
		return fmt.Errorf("get container runtime: %w", err)
	}

	if err := os.MkdirAll(c.workDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating workdir: %w", err)
	}
	kindConfigYamlBytes, err := yaml.Marshal(c.kindClusterConfig)
	if err != nil {
		return fmt.Errorf("failed to process the KinD cluster config as a YAML: %w", err)
	}

	kindconfigPath := filepath.Join(c.workDir, "/kind.yaml")
	if err := os.WriteFile(
		kindconfigPath, kindConfigYamlBytes, os.ModePerm); err != nil {
		return fmt.Errorf("creating kind cluster config: %w", err)
	}

	provider, err := c.getKindProvider()
	if err != nil {
		return err
	}

	// check if cluster already exists with the same name
	clusterExists, err := c.Exists()
	if err != nil {
		return fmt.Errorf("checking cluster exists: %w", err)
	}
	if clusterExists {
		if err := provider.Create(c.name,
			cluster.CreateWithKubeconfigPath(c.kubeconfigPath),
			cluster.CreateWithConfigFile(kindconfigPath),
			cluster.CreateWithDisplayUsage(true),
			cluster.CreateWithDisplaySalutation(true),
			cluster.CreateWithWaitForReady(5*time.Minute),
			cluster.CreateWithRetain(false),
		); err != nil {
			return fmt.Errorf("failed to create the cluster: %w", err)
		}
	}
	return nil
}

// Destroys the KinD cluster if it exists.
func (c *KindCluster) Destroy() error {
	provider, err := c.getKindProvider()
	if err != nil {
		return err
	}
	return provider.Delete(c.name, c.kubeconfigPath)
}

// Load an image from a tar archive into the environment.
func (c *KindCluster) LoadImageFromTar(filePath string) error {
	provider, err := c.getKindProvider()
	if err != nil {
		return err
	}
	nodesList, err := provider.ListInternalNodes(c.name)
	if err != nil {
		return fmt.Errorf("failed to list the nodes of the KinD cluster: %w", err)
	}

	if _, err := os.Stat(filePath); err != nil {
		return err
	}

	for _, node := range nodesList {
		node := node
		if err := loadImageTarIntoNode(filePath, node); err != nil {
			return fmt.Errorf("failed to load the image: %w", err)
		}
	}
	return nil
}

func (c *KindCluster) getKindProvider() (cluster.Provider, error) {
	if c.provider != nil {
		return *c.provider, nil
	}

	var providerOpt cluster.ProviderOption
	switch c.containerRuntime {
	case ContainerRuntimeDocker:
		providerOpt = cluster.ProviderWithDocker()
	case ContainerRuntimePodman:
		providerOpt = cluster.ProviderWithPodman()
	}
	logger := kindcmd.NewLogger()
	return *cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
		providerOpt,
	), nil
}

func loadImageTarIntoNode(imageTarPath string, node nodes.Node) error {
	f, err := os.Open(imageTarPath)
	if err != nil {
		return fmt.Errorf("failed to open the image: %w", err)
	}
	defer f.Close()
	return nodeutils.LoadImageArchive(node, f)
}

func sanitizeKindClusterConfig(conf *kindv1alpha4.Cluster) {
	conf.TypeMeta = kindv1alpha4.TypeMeta{
		Kind:       "Cluster",
		APIVersion: "kind.x-k8s.io/v1alpha4",
	}
}

func defaultKindClusterConfig(conf *kindv1alpha4.Cluster) {
	kindv1alpha4.SetDefaultsCluster(conf)
	if _, err := os.Lstat("/dev/dm-0"); err == nil {
		for i := range conf.Nodes {
			conf.Nodes[i].ExtraMounts = []kindv1alpha4.Mount{
				{
					HostPath:      "/dev/dm-0",
					ContainerPath: "/dev/dm-0",
					Propagation:   kindv1alpha4.MountPropagationHostToContainer,
				},
			}
		}
	}
}
