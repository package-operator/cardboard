package dev

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/go-logr/logr"
	kindv1alpha4 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
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
		defaultCluster := defaultKindConfigCluster()
		c.KindClusterConfig = &defaultCluster
	}
}

func defaultKindConfigCluster() kindv1alpha4.Cluster {
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

	// Needs cluster creation?
	var checkOutput bytes.Buffer
	if err := env.execKindCommand(ctx, &checkOutput, nil, "get", "clusters"); err != nil {
		return fmt.Errorf("getting existing kind clusters: %w", err)
	}

	// Only create cluster if it is not already there.
	createCluster := !strings.Contains(checkOutput.String(), env.Name+"\n")
	if createCluster {
		// Create cluster
		if err := env.execKindCommand(
			ctx, os.Stdout, os.Stderr,
			"create", "cluster",
			"--kubeconfig="+kubeconfigPath,
			"--name="+env.Name,
			"--config="+kindconfigPath,
		); err != nil {
			return fmt.Errorf("creating kind cluster: %w", err)
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
	if err := env.execKindCommand(
		ctx, os.Stdout, os.Stderr,
		"delete", "cluster",
		"--kubeconfig="+path.Join(env.WorkDir, "kubeconfig.yaml"),
		"--name="+env.Name,
	); err != nil {
		return fmt.Errorf("deleting kind cluster: %w", err)
	}
	return nil
}

// Load an image from a tar archive into the environment.
func (env *Environment) LoadImageFromTar(
	ctx context.Context, filePath string) error {
	if err := env.execKindCommand(
		ctx, os.Stdout, os.Stderr,
		"load", "image-archive", filePath,
		"--name="+env.Name,
	); err != nil {
		return fmt.Errorf("loading image archive: %w", err)
	}
	return nil
}

func (env *Environment) RunKindCommand(
	ctx context.Context, stdout, stderr io.Writer, args ...string) error {
	return env.execKindCommand(ctx, stdout, stderr, args...)
}

func (env *Environment) execKindCommand(
	ctx context.Context, stdout, stderr io.Writer, args ...string) error {
	log := logr.FromContextOrDiscard(ctx)
	log.Info("exec: kind " + strings.Join(args, " "))

	kindCmd := exec.CommandContext( //nolint:gosec
		ctx, "kind", args...,
	)
	kindCmd.Env = os.Environ()
	if env.config.ContainerRuntime == "podman" {
		kindCmd.Env = append(kindCmd.Env, "KIND_EXPERIMENTAL_PROVIDER=podman")
	}
	kindCmd.Stdout = stdout
	kindCmd.Stderr = stderr
	return kindCmd.Run()
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
