package dev

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Load objects from given folder paths and applies them into the cluster.
type ClusterLoadObjectsFromFolders []string

func (l ClusterLoadObjectsFromFolders) Init(
	ctx context.Context, cluster *Cluster) error {
	return cluster.CreateAndWaitFromFolders(ctx, l)
}

// Load objects from given file paths and applies them into the cluster.
type ClusterLoadObjectsFromFiles []string

func (l ClusterLoadObjectsFromFiles) Init(
	ctx context.Context, cluster *Cluster) error {
	return cluster.CreateAndWaitFromFiles(ctx, l)
}

// Load objects from the given http urls and applies them into the cluster.
type ClusterLoadObjectsFromHttp []string

func (l ClusterLoadObjectsFromHttp) Init(
	ctx context.Context, cluster *Cluster) error {
	return cluster.CreateAndWaitFromHttp(ctx, l)
}

// Load objects from the given client.Object and applies them into the cluster.
type ClusterLoadObjectFromClientObject struct {
	client.Object
}

func (l ClusterLoadObjectFromClientObject) Init(
	ctx context.Context, cluster *Cluster) error {
	return cluster.CreateAndWaitForReadiness(ctx, l.Object)
}

// Adds the helm repository, updates repository cache and installs a helm package.
type ClusterHelmInstall struct {
	RepoName, RepoURL, PackageName, Namespace, ReleaseName string
	SetVars                                                []string
}

func (h ClusterHelmInstall) Init(
	ctx context.Context, cluster *Cluster) error {
	if err := cluster.Helm.HelmRepoAdd(ctx, h.RepoName, h.RepoURL); err != nil {
		return err
	}
	if err := cluster.Helm.HelmRepoUpdate(ctx); err != nil {
		return err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: h.Namespace,
		},
	}
	if err := cluster.CtrlClient.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("creating namespace for helm: %w", err)
	}

	return cluster.Helm.HelmInstall(ctx, cluster, h.RepoName, h.PackageName, h.ReleaseName, h.Namespace, h.SetVars)
}
