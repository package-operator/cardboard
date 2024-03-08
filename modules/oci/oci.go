package oci

import (
	"context"
	"fmt"

	"pkg.package-operator.run/cardboard/kubeutils"
	"pkg.package-operator.run/cardboard/sh"
)

type shRunner interface {
	Run(cmd string, args ...string) error
}

const (
	ociTarFilename = "container.oci.tar"
	ociDigestFile  = "digestfile"
)

// Open Container Image.
type OCI struct {
	tag              string
	containerFile    string
	workDir          string
	cranePush        bool
	containerRuntime kubeutils.ContainerRuntime
	runner           shRunner
}

type Option interface {
	ApplyToOCI(oci *OCI)
}

type WithContainerFile string

func (cf WithContainerFile) ApplyToOCI(oci *OCI) {
	oci.containerFile = string(cf)
}

type WithCranePush struct{}

func (cf WithCranePush) ApplyToOCI(oci *OCI) {
	oci.cranePush = true
}

func NewOCI(tag, workDir string, opts ...Option) *OCI {
	oci := &OCI{
		tag:     tag,
		workDir: workDir,
		runner:  sh.New(sh.WithWorkDir(workDir)),
	}
	for _, opt := range opts {
		opt.ApplyToOCI(oci)
	}
	return oci
}

func (oci *OCI) Load(path string) error {
	cr, err := kubeutils.ContainerRuntimeOrDetect(oci.containerRuntime)
	if err != nil {
		return err
	}
	return sh.New().Run(string(cr), "load", "-i", path)
}

func (oci *OCI) ID() string {
	return fmt.Sprintf("pkg.package-operator.run/cardboard/modules/oci.OCI{name:%s}", oci.tag)
}

// Returns a Build dependency.
func (oci *OCI) Run(_ context.Context) error {
	return oci.Build()
}

// Build the image.
func (oci *OCI) Build() error {
	cr, err := kubeutils.ContainerRuntimeOrDetect(oci.containerRuntime)
	if err != nil {
		return err
	}

	buildCmdArgs := []string{"build"}
	buildCmdArgs = append(buildCmdArgs, "-t", oci.tag)

	if oci.containerFile != "" {
		buildCmdArgs = append(buildCmdArgs, "-f", oci.containerFile)
	}
	buildCmdArgs = append(buildCmdArgs, oci.workDir)
	if err := oci.runner.Run(string(cr), buildCmdArgs...); err != nil {
		return err
	}

	imgSaveArgs := []string{
		"image", "save",
		"-o", ociTarFilename, oci.tag,
	}
	if err := oci.runner.Run(string(cr), imgSaveArgs...); err != nil {
		return err
	}
	return nil
}

// Push the image.
func (oci *OCI) Push() error {
	if oci.cranePush {
		return oci.pushWithCrane()
	}
	return oci.pushWithCR()
}

func (oci *OCI) pushWithCrane() error {
	args := []string{"push", ociTarFilename, oci.tag}
	if err := oci.runner.Run("crane", args...); err != nil {
		return err
	}

	return nil
}

func (oci *OCI) pushWithCR() error {
	cr, err := kubeutils.ContainerRuntimeOrDetect(oci.containerRuntime)
	if err != nil {
		return err
	}

	args := []string{"push"}
	if cr == kubeutils.ContainerRuntimePodman {
		args = append(args, "--digestfile="+ociDigestFile)
	}
	args = append(args, oci.tag)
	if err := oci.runner.Run(string(cr), args...); err != nil {
		return err
	}

	return nil
}
