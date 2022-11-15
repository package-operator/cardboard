package dev

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"time"

	"sigs.k8s.io/yaml"

	kindv1alpha4 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	kindcmd "sigs.k8s.io/kind/pkg/cmd"
)

type EnvironmentConfig struct {
	// Cluster initializers prepare a cluster for use.
	ClusterInitializers []ClusterInitializer
	// Container runtime to use
	ContainerRuntime  ContainerRuntime
	NewCluster        NewClusterFunc
	ClusterOptions    []ClusterOption
	KindClusterConfig *kindv1alpha4.Cluster
}

// Apply default configuration.
func (c *EnvironmentConfig) Default() {
	if len(c.ContainerRuntime) == 0 {
		c.ContainerRuntime = ContainerRuntimeAuto
	}
	if c.NewCluster == nil {
		c.NewCluster = NewCluster
	}
	if c.KindClusterConfig == nil {
		defaultCluster := defaultKindClusterConfig()
		c.KindClusterConfig = &defaultCluster
	}
}

func sanitizeKindClusterConfig(conf *kindv1alpha4.Cluster) {
	conf.TypeMeta = kindv1alpha4.TypeMeta{
		Kind:       "Cluster",
		APIVersion: "kind.x-k8s.io/v1alpha4",
	}
}

func defaultKindClusterConfig() kindv1alpha4.Cluster {
	cluster := kindv1alpha4.Cluster{
		TypeMeta: kindv1alpha4.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "kind.x-k8s.io/v1alpha4",
		},
		Nodes: []kindv1alpha4.Node{
			{
				Role: kindv1alpha4.ControlPlaneRole,
			},
		},
	}

	if _, err := os.Lstat("/dev/dm-0"); err == nil {
		cluster.Nodes[0].ExtraMounts = []kindv1alpha4.Mount{
			{
				HostPath:      "/dev/dm-0",
				ContainerPath: "/dev/dm-0",
				Propagation:   kindv1alpha4.MountPropagationHostToContainer,
			},
		}
	}
	kindv1alpha4.SetDefaultsCluster(&cluster)
	sanitizeKindClusterConfig(&cluster)
	return cluster
}

type EnvironmentOption interface {
	ApplyToEnvironmentConfig(c *EnvironmentConfig)
}

type NewClusterFunc func(kubeconfigPath string, opts ...ClusterOption) (*Cluster, error)

type ClusterInitializer interface {
	Init(ctx context.Context, cluster *Cluster) error
}

// Environment represents a development environment.
type Environment struct {
	Name string
	// Working directory of the environment.
	// Temporary files/kubeconfig etc. will be stored here.
	WorkDir string
	Cluster *Cluster
	config  EnvironmentConfig
}

// Creates a new development environment.
func NewEnvironment(name, workDir string, opts ...EnvironmentOption) *Environment {
	env := &Environment{
		Name:    name,
		WorkDir: workDir,
	}
	for _, opt := range opts {
		opt.ApplyToEnvironmentConfig(&env.config)
	}
	env.config.Default()
	return env
}

// Initializes the environment and prepares it for use.
func (env *Environment) Init(ctx context.Context) error {
	if err := env.setContainerRuntime(); err != nil {
		return err
	}

	if env.config.KindClusterConfig == nil {
		return fmt.Errorf("no KinD cluster config found")
	}

	if err := os.MkdirAll(env.WorkDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating workdir: %w", err)
	}

	kindConfigYamlBytes, err := yaml.Marshal(env.config.KindClusterConfig)
	if err != nil {
		return fmt.Errorf("failed to process the KinD cluster config as a YAML: %w", err)
	}

	kubeconfigPath := path.Join(env.WorkDir, "kubeconfig.yaml")
	kindconfigPath := path.Join(env.WorkDir, "/kind.yaml")
	if err := ioutil.WriteFile(
		kindconfigPath, kindConfigYamlBytes, os.ModePerm); err != nil {
		return fmt.Errorf("creating kind cluster config: %w", err)
	}

	provider, err := env.getKindProvider()
	if err != nil {
		return err
	}

	// check if cluster already exists with the same name
	createCluster := true
	existingKindClusters, err := provider.List()
	if err != nil {
		return fmt.Errorf("failed to fetch the existing KinD clusters: %w", err)
	}
	for _, cluster := range existingKindClusters {
		if env.Name == cluster {
			createCluster = false
			break
		}
	}

	if createCluster {
		if err := provider.Create(env.Name,
			cluster.CreateWithKubeconfigPath(kubeconfigPath),
			cluster.CreateWithConfigFile(kindconfigPath),
			cluster.CreateWithDisplayUsage(true),
			cluster.CreateWithDisplaySalutation(true),
			cluster.CreateWithWaitForReady(5*time.Minute),
			cluster.CreateWithRetain(false),
		); err != nil {
			return fmt.Errorf("failed to create the cluster: %w", err)
		}
	}

	// Create _all_ the clients
	cluster, err := env.config.NewCluster(
		env.WorkDir, append(env.config.ClusterOptions, WithKubeconfigPath(kubeconfigPath))...)
	if err != nil {
		return fmt.Errorf("creating k8s clients: %w", err)
	}
	env.Cluster = cluster

	// Run ClusterInitializers
	if createCluster {
		for _, initializer := range env.config.ClusterInitializers {
			if err := initializer.Init(ctx, cluster); err != nil {
				return fmt.Errorf("running cluster initializer: %w", err)
			}
		}
	}

	return nil
}

// Destroy/Teardown the development environment.
func (env *Environment) Destroy(ctx context.Context) error {
	provider, err := env.getKindProvider()
	if err != nil {
		return err
	}

	kubeConfigPath := path.Join(env.WorkDir, "kubeconfig.yaml")
	if err := provider.Delete(env.Name, kubeConfigPath); err != nil {
		return fmt.Errorf("failed to delete the cluster: %w", err)
	}
	return nil
}

// Load an image from a tar archive into the environment.
func (env *Environment) LoadImageFromTar(filePath string) error {
	provider, err := env.getKindProvider()
	if err != nil {
		return err
	}
	nodesList, err := provider.ListInternalNodes(env.Name)
	if err != nil {
		return fmt.Errorf("failed to list the nodes of the KinD cluster: %w", err)
	}

	if _, err := os.Stat(filePath); err != nil {
		return err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open the image: %w", err)
	}
	defer f.Close()
	for _, node := range nodesList {
		if err := nodeutils.LoadImageArchive(node, f); err != nil {
			return fmt.Errorf("failed to load the image: %w", err)
		}
	}
	return nil
}

func (env *Environment) setContainerRuntime() error {
	if env.config.ContainerRuntime == ContainerRuntimeAuto {
		cr, err := DetectContainerRuntime()
		if err != nil {
			return err
		}
		env.config.ContainerRuntime = cr
	}
	return nil
}

func DetectContainerRuntime() (ContainerRuntime, error) {
	if _, err := exec.LookPath("podman"); err == nil {
		return ContainerRuntimePodman, nil
	} else if !errors.Is(err, exec.ErrNotFound) {
		return "", fmt.Errorf("looking up podman executable: %w", err)
	}

	if _, err := exec.LookPath("docker"); err == nil {
		return ContainerRuntimeDocker, nil
	} else if !errors.Is(err, exec.ErrNotFound) {
		return "", fmt.Errorf("looking up docker executable: %w", err)
	}
	return "", fmt.Errorf("could not detect container runtime")
}

func (env *Environment) getKindProvider() (cluster.Provider, error) {
	var providerOpt cluster.ProviderOption
	switch env.config.ContainerRuntime {
	case ContainerRuntimeDocker:
		providerOpt = cluster.ProviderWithDocker()
	case ContainerRuntimePodman:
		providerOpt = cluster.ProviderWithPodman()
	case ContainerRuntimeAuto:
		if err := env.setContainerRuntime(); err != nil {
			return cluster.Provider{}, fmt.Errorf("failed to auto-set the container runtime: %w", err)
		}
		return env.getKindProvider()
	default:
		return cluster.Provider{}, fmt.Errorf("unknown container runtime found")
	}
	logger := kindcmd.NewLogger()
	return *cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
		providerOpt,
	), nil

}
