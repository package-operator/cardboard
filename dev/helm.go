package dev

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
)

type HelmConfig struct {
	WorkDir        string
	Kubeconfig     string
	Stdout, Stderr io.Writer
}

func (c *HelmConfig) Default() {
	if c.Stdout == nil {
		c.Stdout = os.Stdout
	}
	if c.Stderr == nil {
		c.Stderr = os.Stderr
	}
}

type HelmOption interface {
	ApplyToHelmConfig(c *HelmConfig)
}

type Helm struct {
	HelmConfig
}

func NewHelm(workDir, kubeconfig string, opts ...HelmOption) *Helm {
	h := &Helm{
		HelmConfig: HelmConfig{
			WorkDir:    workDir,
			Kubeconfig: kubeconfig,
		},
	}
	for _, opt := range opts {
		opt.ApplyToHelmConfig(&h.HelmConfig)
	}
	h.HelmConfig.Default()
	return h
}

// Wrapper arround "helm repo add"
func (h *Helm) HelmRepoAdd(
	ctx context.Context, repoName, repoURL string,
) error {
	if err := h.execHelmCommand(
		ctx, os.Stdout, os.Stderr,
		"repo", "add", repoName, repoURL,
	); err != nil {
		return fmt.Errorf("helm repo add: %w", err)
	}
	return nil
}

// Wrapper arround "helm repo update"
func (h *Helm) HelmRepoUpdate(ctx context.Context) error {
	if err := h.execHelmCommand(
		ctx, os.Stdout, os.Stderr,
		"repo", "update",
	); err != nil {
		return fmt.Errorf("helm repo update: %w", err)
	}
	return nil
}

// Wrapper arround "helm install"
func (h *Helm) HelmInstall(
	ctx context.Context, cluster *Cluster,
	repoName, packageName, releaseName, namespace string,
	setVars []string,
) error {
	installFlags := []string{
		"install", releaseName, repoName + "/" + packageName,
	}
	if len(namespace) > 0 {
		installFlags = append(installFlags, "-n", namespace)
	}
	for _, s := range setVars {
		installFlags = append(installFlags, "--set", s)
	}
	if err := h.execHelmCommand(
		ctx, os.Stdout, os.Stderr,
		installFlags...,
	); err != nil {
		return fmt.Errorf("helm repo update: %w", err)
	}
	return nil
}

func (h *Helm) execHelmCommand(
	ctx context.Context, stdout, stderr io.Writer, args ...string,
) error {
	helmCacheDir := path.Join(h.WorkDir, "helm/cache")
	helmConfigDir := path.Join(h.WorkDir, "helm/config")
	helmDataDir := path.Join(h.WorkDir, "helm/data")

	for _, dir := range []string{helmCacheDir, helmConfigDir, helmDataDir} {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("create helm dir %s: %w", dir, err)
		}
	}

	helmCmd := exec.CommandContext( //nolint:gosec
		ctx, "helm", args...,
	)
	helmCmd.Env = os.Environ()
	helmCmd.Env = append(
		helmCmd.Env,
		"KUBECONFIG="+h.Kubeconfig,
		"HELM_CACHE_HOME="+helmCacheDir,
		"HELM_CONFIG_HOME="+helmConfigDir,
		"HELM_DATA_HOME="+helmDataDir,
	)
	helmCmd.Stdout = stdout
	helmCmd.Stderr = stderr
	return helmCmd.Run()
}
