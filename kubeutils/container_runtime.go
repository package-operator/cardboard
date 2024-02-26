package kubeutils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
)

type ContainerRuntime string

const (
	ContainerRuntimePodman ContainerRuntime = "podman"
	ContainerRuntimeDocker ContainerRuntime = "docker"
)

func isValidCR(cr ContainerRuntime) bool {
	return slices.Contains([]ContainerRuntime{ContainerRuntimePodman, ContainerRuntimeDocker}, cr)
}

func ContainerRuntimeOrDetect(cr ContainerRuntime) (ContainerRuntime, error) {
	if isValidCR(cr) {
		return cr, nil
	}

	crStr, crStrOK := os.LookupEnv("CARDBOARD_CONTAINER_RUNTIME")
	if crStrOK {
		cr = ContainerRuntime(crStr)
		if !isValidCR(cr) {
			return cr, errors.New("invalid value for CARDBOARD_CONTAINER_RUNTIME")
		}

		return cr, nil
	}

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
	return "", errors.New("could not detect container runtime")
}
