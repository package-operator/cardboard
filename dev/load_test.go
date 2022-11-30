package dev

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestLoadAndConvertIntoObject(t *testing.T) {
	testScheme := runtime.NewScheme()
	if err := appsv1.AddToScheme(testScheme); err != nil {
		panic(err)
	}

	deployment := &appsv1.Deployment{}
	err := LoadAndConvertIntoObject(testScheme, "testdata/deployment.yaml", deployment)
	require.NoError(t, err)
	assert.Equal(t, "test-deployment", deployment.Name)
	assert.Equal(t, "test-namespace", deployment.Namespace)
	expectedAnnotations := map[string]string{}
	expectedAnnotations["test-annotation"] = "test-value"
	assert.Equal(t, expectedAnnotations, deployment.Annotations)
	expectedLabels := map[string]string{}
	expectedLabels["test-label"] = "test-value"
	assert.Equal(t, expectedLabels, deployment.Labels)
	assert.Equal(t, 1, int(*deployment.Spec.Replicas))
	assert.Equal(t, expectedLabels, deployment.Spec.Template.Labels)
	expectedContainers := []corev1.Container{
		{Name: "test-container", Image: "test-image:1.2.3"}}
	assert.Equal(t, expectedContainers,
		deployment.Spec.Template.Spec.Containers)
}

func TestLoadAndUnmarshalIntoObject(t *testing.T) {
	deployment := &appsv1.Deployment{}
	err := LoadAndUnmarshalIntoObject("testdata/deployment.yaml", deployment)
	require.NoError(t, err)
	assert.Equal(t, "test-deployment", deployment.Name)
	assert.Equal(t, "test-namespace", deployment.Namespace)
	expectedAnnotations := map[string]string{}
	expectedAnnotations["test-annotation"] = "test-value"
	assert.Equal(t, expectedAnnotations, deployment.Annotations)
	expectedLabels := map[string]string{}
	expectedLabels["test-label"] = "test-value"
	assert.Equal(t, expectedLabels, deployment.Labels)
	assert.Equal(t, 1, int(*deployment.Spec.Replicas))
	assert.Equal(t, expectedLabels, deployment.Spec.Template.Labels)
	expectedContainers := []corev1.Container{
		{Name: "test-container", Image: "test-image:1.2.3"}}
	assert.Equal(t, expectedContainers,
		deployment.Spec.Template.Spec.Containers)
}
