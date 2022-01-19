package dev

import "github.com/go-logr/logr"

const (
	olmVersion = "0.19.1"
)

func ExampleEnvironment() {
	log := logr.Discard()

	_ = NewEnvironment(
		"cheese", ".cache/dev-env/cheese",
		WithLogger(log),
		WithContainerRuntime(Podman),
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
}
