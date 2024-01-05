package kind

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

	"pkg.package-operator.run/cardboard/kubeutils"
	"pkg.package-operator.run/cardboard/modules/kubeclients"
	"pkg.package-operator.run/cardboard/run"
)

const defaultCacheDirectory = ".cache"

type Cluster struct {
	name, workDir    string
	kubeconfigPath   string
	containerRuntime kubeutils.ContainerRuntime
	clusterConfig    *kindv1alpha4.Cluster

	clients  *kubeclients.KubeClients
	provider *cluster.Provider
}

type ClusterOption interface {
	ApplyToCluster(kc *Cluster)
}

func NewCluster(name string, opts ...ClusterOption) *Cluster {
	kc := &Cluster{
		name:    name,
		workDir: filepath.Join(defaultCacheDirectory, "clusters", name),
	}
	kc.kubeconfigPath = filepath.Join(kc.workDir, "kubeconfig.yaml")
	for _, opt := range opts {
		opt.ApplyToCluster(kc)
	}
	if kc.clusterConfig == nil {
		kc.clusterConfig = &kindv1alpha4.Cluster{}
	}
	sanitizeClusterConfig(kc.clusterConfig)
	defaultClusterConfig(kc.clusterConfig)
	return kc
}

func (c *Cluster) KubeconfigPath() string {
	return c.kubeconfigPath
}

func (c *Cluster) ID() string {
	return fmt.Sprintf("pkg.package-operator.run/cardboard/modules.Cluster{name:%s}", c.name)
}

// Check if the cluster already exists.
func (c *Cluster) Exists() (bool, error) {
	provider, err := c.getKindProvider()
	if err != nil {
		return false, err
	}
	existingClusters, err := provider.List()
	if err != nil {
		return false, fmt.Errorf("failed to fetch the existing KinD clusters: %w", err)
	}
	for _, cluster := range existingClusters {
		if c.name == cluster {
			return true, nil
		}
	}
	return false, nil
}

func (c *Cluster) Clients() (*kubeclients.KubeClients, error) {
	if c.clients != nil {
		return c.clients, nil
	}
	c.clients = kubeclients.NewKubeClients(c.kubeconfigPath)
	if err := c.clients.Run(context.Background()); err != nil {
		return nil, err
	}
	return c.clients, nil
}

// Returns a Create dependency for stetting up pre-requisites.
func (c *Cluster) CreateDep() run.Dependency {
	return run.Meth(c, c.Create)
}

// Returns a Destroy dependency for stetting up pre-requisites.
func (c *Cluster) DestroyDep() run.Dependency {
	return run.Meth(c, c.Destroy)
}

// Creates the KinD cluster if it does not exist.
func (c *Cluster) Create() error {
	var err error
	c.containerRuntime, err = c.containerRuntime.Get()
	if err != nil {
		return fmt.Errorf("get container runtime: %w", err)
	}

	if err := os.MkdirAll(c.workDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating workdir: %w", err)
	}
	kindConfigYamlBytes, err := yaml.Marshal(c.clusterConfig)
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
func (c *Cluster) Destroy() error {
	provider, err := c.getKindProvider()
	if err != nil {
		return err
	}
	return provider.Delete(c.name, c.kubeconfigPath)
}

// Load an image from a tar archive into the environment.
func (c *Cluster) LoadImageFromTar(filePath string) error {
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

func (c *Cluster) getKindProvider() (cluster.Provider, error) {
	if c.provider != nil {
		return *c.provider, nil
	}

	var providerOpt cluster.ProviderOption
	switch c.containerRuntime {
	case kubeutils.ContainerRuntimeDocker:
		providerOpt = cluster.ProviderWithDocker()
	case kubeutils.ContainerRuntimePodman:
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

func sanitizeClusterConfig(conf *kindv1alpha4.Cluster) {
	conf.TypeMeta = kindv1alpha4.TypeMeta{
		Kind:       "Cluster",
		APIVersion: "kind.x-k8s.io/v1alpha4",
	}
}

func defaultClusterConfig(conf *kindv1alpha4.Cluster) {
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
