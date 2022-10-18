package dev

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

const (
	olmVersion = "0.19.1"
)

func ExampleEnvironment() {
	log := logr.Discard()

	env := NewEnvironment(
		"cheese", ".cache/dev-env/cheese",
		WithContainerRuntime(ContainerRuntimePodman),
		WithClusterInitializers{
			ClusterLoadObjectsFromFiles{
				"config/crd01.yaml",
				"config/crd02.yaml",
				"config/deploy.yaml",
			},
			ClusterLoadObjectsFromFolders{
				"config/logging-stack",
			},
			ClusterLoadObjectsFromHttp{
				// Install OLM.
				"https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v" + olmVersion + "/crds.yaml",
				"https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v" + olmVersion + "/olm.yaml",
			},
			ClusterHelmInstall{
				RepoName:    "prometheus-community",
				RepoURL:     "https://prometheus-community.github.io/helm-charts",
				PackageName: "kube-prometheus-stack",
				ReleaseName: "prometheus",
				Namespace:   "monitoring",
				SetVars: []string{
					"grafana.enabled=false",
					"kubeStateMetrics.enabled=false",
					"nodeExporter.enabled=false",
				},
			},
		},
	)
	ctx := logr.NewContext(context.Background(), log)
	if err := env.Init(ctx); err != nil {
		// handle error
	}
}

func TestEnvironmentConfig_Default(t *testing.T) {
	var c EnvironmentConfig
	c.Default()

	assertEnvironmentConfigDefaults(t, c)
}

func TestEnvironment_NewEnvironment(t *testing.T) {
	e := NewEnvironment("cheese", "./cheese")

	assertEnvironmentConfigDefaults(t, e.config)
}

func assertEnvironmentConfigDefaults(t *testing.T, c EnvironmentConfig) {
	t.Helper()

	assert.Equal(t, ContainerRuntimeAuto, c.ContainerRuntime)
	assert.NotNil(t, c.NewCluster)
}
