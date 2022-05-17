package dev

import (
	"io"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
)

type WithLogger logr.Logger

func (l WithLogger) ApplyToEnvironmentConfig(c *EnvironmentConfig) {
	c.Logger = logr.Logger(l)
}

func (l WithLogger) ApplyToClusterConfig(c *ClusterConfig) {
	c.Logger = logr.Logger(l)
}

func (l WithLogger) ApplyToWaiterConfig(c *WaiterConfig) {
	c.Logger = logr.Logger(l)
}

type WithInterval time.Duration

func (i WithInterval) ApplyToWaiterConfig(c *WaiterConfig) {
	c.Interval = time.Duration(i)
}

type WithTimeout time.Duration

func (t WithTimeout) ApplyToWaiterConfig(c *WaiterConfig) {
	c.Timeout = time.Duration(t)
}

type WithSchemeBuilder runtime.SchemeBuilder

func (sb WithSchemeBuilder) ApplyToClusterConfig(c *ClusterConfig) {
	c.SchemeBuilder = runtime.SchemeBuilder(sb)
}

type WithNewWaiterFunc NewWaiterFunc

func (f WithNewWaiterFunc) ApplyToClusterConfig(c *ClusterConfig) {
	c.NewWaiter = NewWaiterFunc(f)
}

type WithWaitOptions []WaitOption

func (opts WithWaitOptions) ApplyToClusterConfig(c *ClusterConfig) {
	c.WaitOptions = []WaitOption(opts)
}

type WithNewHelmFunc NewHelmFunc

func (f WithNewHelmFunc) ApplyToClusterConfig(c *ClusterConfig) {
	c.NewHelm = NewHelmFunc(f)
}

type WithHelmOptions []HelmOption

func (opts WithHelmOptions) ApplyToClusterConfig(c *ClusterConfig) {
	c.HelmOptions = []HelmOption(opts)
}

type WithStdout struct{ io.Writer }

func (w WithStdout) ApplyToHelmConfig(c *HelmConfig) {
	c.Stdout = io.Writer(w)
}

type WithStderr struct{ io.Writer }

func (w WithStderr) ApplyToHelmConfig(c *HelmConfig) {
	c.Stderr = io.Writer(w)
}

type WithClusterInitializers []ClusterInitializer

func (i WithClusterInitializers) ApplyToEnvironmentConfig(c *EnvironmentConfig) {
	c.ClusterInitializers = append(c.ClusterInitializers, i...)
}

type ContainerRuntime string

const (
	ContainerRuntimePodman ContainerRuntime = "podman"
	ContainerRuntimeDocker ContainerRuntime = "docker"
	ContainerRuntimeAuto   ContainerRuntime = "auto" // auto detect
)

type WithContainerRuntime ContainerRuntime

func (cr WithContainerRuntime) ApplyToEnvironmentConfig(c *EnvironmentConfig) {
	c.ContainerRuntime = ContainerRuntime(cr)
}

type WithNewClusterFunc NewClusterFunc

func (f WithNewClusterFunc) ApplyToEnvironmentConfig(c *EnvironmentConfig) {
	c.NewCluster = NewClusterFunc(f)
}

type WithClusterOptions []ClusterOption

func (opts WithClusterOptions) ApplyToEnvironmentConfig(c *EnvironmentConfig) {
	c.ClusterOptions = opts
}

type WithKubeconfigPath string

func (kubeconfig WithKubeconfigPath) ApplyToClusterConfig(c *ClusterConfig) {
	c.Kubeconfig = string(kubeconfig)
}

type WithNewRestConfigFunc NewRestConfigFunc

func (f WithNewRestConfigFunc) ApplyToClusterConfig(c *ClusterConfig) {
	c.NewRestConfig = NewRestConfigFunc(f)
}

type WithNewCtrlClientFunc NewCtrlClientFunc

func (f WithNewCtrlClientFunc) ApplyToClusterConfig(c *ClusterConfig) {
	c.NewCtrlClient = NewCtrlClientFunc(f)
}
