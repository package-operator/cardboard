package modules

import (
	"fmt"

	"pkg.package-operator.run/cardboard/kubeutils"
	"pkg.package-operator.run/cardboard/run"
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
	name             string
	registries       []string
	tags             []string
	containerFile    string
	workDir          string
	containerRuntime kubeutils.ContainerRuntime
	runner           shRunner
}

type OCIOption interface {
	ApplyToOCI(oci *OCI)
}

type WithTags []string

func (t WithTags) ApplyToOCI(oci *OCI) {
	oci.tags = t
}

type WithRegistries []string

func (t WithRegistries) ApplyToOCI(oci *OCI) {
	oci.registries = t
}

type WithContainerFile string

func (cf WithContainerFile) ApplyToOCI(oci *OCI) {
	oci.containerFile = string(cf)
}

func NewOCI(name, workDir string, opts ...OCIOption) *OCI {
	oci := &OCI{
		name:    name,
		workDir: workDir,
		runner:  sh.New(sh.WithWorkDir(workDir)),
	}
	for _, opt := range opts {
		opt.ApplyToOCI(oci)
	}
	if len(oci.tags) == 0 {
		oci.tags = []string{"latest"}
	}
	return oci
}

func (oci *OCI) ID() string {
	return fmt.Sprintf("pkg.package-operator.run/cardboard/modules.OCI{name:%s}", oci.name)
}

// Returns a Build dependency.
func (oci *OCI) BuildDep() run.Dependency {
	return run.Meth(oci, oci.Build)
}

// Returns a Push dependency.
func (oci *OCI) PushDep() run.Dependency {
	return run.Meth(oci, oci.Push)
}

// Build the image.
func (oci *OCI) Build() error {
	cr, err := oci.containerRuntime.Get()
	if err != nil {
		return err
	}

	buildCmdArgs := []string{"build"}
	tags := registryNameTags(oci.name, oci.registries, oci.tags)
	for _, t := range tags {
		buildCmdArgs = append(buildCmdArgs, "-t", t)
	}
	if oci.containerFile != "" {
		buildCmdArgs = append(buildCmdArgs, "-f", oci.containerFile)
	}
	buildCmdArgs = append(buildCmdArgs, oci.workDir)
	if err := oci.runner.Run(string(cr), buildCmdArgs...); err != nil {
		return err
	}

	imgSaveArgs := []string{
		"image", "save",
		"-o", ociTarFilename, tags[0],
	}
	if err := oci.runner.Run(string(cr), imgSaveArgs...); err != nil {
		return err
	}
	return nil
}

// Push the image.
func (oci *OCI) Push() error {
	cr, err := oci.containerRuntime.Get()
	if err != nil {
		return err
	}

	tags := registryNameTags(oci.name, oci.registries, oci.tags)
	for _, t := range tags {
		args := []string{"push"}
		if cr == kubeutils.ContainerRuntimePodman {
			args = append(args, "--digestfile="+ociDigestFile)
		}
		args = append(args, t)

		if err := oci.runner.Run(string(cr), args...); err != nil {
			return err
		}
	}
	return nil
}

func registryNameTags(name string, registries []string, tags []string) []string {
	var out []string
	for _, r := range registries {
		for _, t := range tags {
			out = append(out, fmt.Sprintf("%s/%s:%s", r, name, t))
		}
	}
	return out
}
