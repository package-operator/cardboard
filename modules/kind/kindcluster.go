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
)

const defaultCacheDirectory = ".cache"

// Represents a KinD cluster.
type Cluster struct {
	name, workDir    string
	kubeconfigPath   string
	containerRuntime kubeutils.ContainerRuntime
	clusterConfig    *kindv1alpha4.Cluster

	clients            *kubeclients.KubeClients
	kubeClientsOptions []kubeclients.Option
	provider           *cluster.Provider
	initializers       []ClusterInitializer
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

func (c *Cluster) Kubeconfig(internal bool) (string, error) {
	provider, err := providerOrFromCR(c.provider, c.containerRuntime)
	if err != nil {
		return "", err
	}

	return provider.KubeConfig(c.name, internal)
}

func (c *Cluster) KubeconfigPath() (string, error) {
	return filepath.Abs(c.kubeconfigPath)
}

func (c *Cluster) ID() string {
	return fmt.Sprintf("pkg.package-operator.run/cardboard/modules/kind.Cluster{name:%s}", c.name)
}

func (c *Cluster) IPv4() (string, error) {
	clusterNodes, err := c.Nodes(false)
	if err != nil {
		// TODO: augment error
		return "", err
	}
	for _, node := range clusterNodes {
		role, err := node.Role()
		if err != nil {
			// TODO: augment error
			return "", err
		}
		if role == "control-plane" {
			ipv4, _, err := node.IP()
			if err != nil {
				// TODO: augment error
				return "", err
			}
			return ipv4, nil
		}
	}
	return "", fmt.Errorf("can't find control plane node for cluster %s", c.Name())
}

func (c *Cluster) Name() string { return c.name }

func (c *Cluster) Nodes(internal bool) ([]nodes.Node, error) {
	provider, err := providerOrFromCR(c.provider, c.containerRuntime)
	if err != nil {
		return []nodes.Node{}, err
	}
	if internal {
		return provider.ListInternalNodes(c.name)
	}
	return provider.ListNodes(c.name)
}

func (c *Cluster) ExportLogs(path string) error {
	provider, err := providerOrFromCR(c.provider, c.containerRuntime)
	if err != nil {
		return err
	}
	return provider.CollectLogs(c.name, path)
}

// Check if the cluster already exists.
func (c *Cluster) Exists() (bool, error) {
	provider, err := providerOrFromCR(c.provider, c.containerRuntime)
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
	c.clients = kubeclients.NewKubeClients(c.kubeconfigPath, c.kubeClientsOptions...)
	if err := c.clients.Run(context.Background()); err != nil {
		return nil, err
	}
	return c.clients, nil
}

// Returns a Create dependency to ensure the cluster is setup.
func (c *Cluster) Run(ctx context.Context) error {
	return c.Create(ctx)
}

// Creates the KinD cluster if it does not exist.
func (c *Cluster) Create(ctx context.Context) error {
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

	provider, err := providerOrFromCR(c.provider, c.containerRuntime)
	if err != nil {
		return err
	}

	// check if cluster already exists with the same name
	clusterExists, err := c.Exists()
	if err != nil {
		return fmt.Errorf("checking cluster exists: %w", err)
	}
	if clusterExists {
		return nil
	}

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
	if _, err := c.Clients(); err != nil {
		return err
	}
	for _, init := range c.initializers {
		if err := init.Init(ctx, c); err != nil {
			return fmt.Errorf("initializing cluster: %w", err)
		}
	}
	return nil
}

// Destroys the KinD cluster if it exists.
func (c *Cluster) Destroy(_ context.Context) error {
	provider, err := providerOrFromCR(c.provider, c.containerRuntime)
	if err != nil {
		return err
	}
	return provider.Delete(c.name, c.kubeconfigPath)
}

// Load an image from a tar archive into the environment.
func (c *Cluster) LoadImageFromTar(filePath string) error {
	provider, err := providerOrFromCR(c.provider, c.containerRuntime)
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

func providerOrFromCR(p *cluster.Provider, cr kubeutils.ContainerRuntime) (*cluster.Provider, error) {
	if p != nil {
		return p, nil
	}

	cr, err := kubeutils.ContainerRuntimeOrDetect(cr)
	if err != nil {
		return nil, err
	}

	var providerOpt cluster.ProviderOption
	switch cr {
	case kubeutils.ContainerRuntimeDocker:
		providerOpt = cluster.ProviderWithDocker()
	case kubeutils.ContainerRuntimePodman:
		providerOpt = cluster.ProviderWithPodman()
	default:
		panic("unknown cr")
	}
	return cluster.NewProvider(cluster.ProviderWithLogger(kindcmd.NewLogger()), providerOpt), nil
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
	// https://github.com/kubernetes-sigs/kind/issues/2411
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
