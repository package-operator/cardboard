package modules

import (
	"errors"
	"fmt"
	"os/exec"
)

type ContainerRuntime string

const (
	ContainerRuntimePodman ContainerRuntime = "podman"
	ContainerRuntimeDocker ContainerRuntime = "docker"
	ContainerRuntimeAuto   ContainerRuntime = "auto" // auto detect
)

func (cr ContainerRuntime) Get() (ContainerRuntime, error) {
	switch cr {
	case ContainerRuntimePodman, ContainerRuntimeDocker:
		return cr, nil
	}
	return DetectContainerRuntime()
}

// Detects the available container runtime, priotizes podman before docker.
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
