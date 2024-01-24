package wait

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWaiterConfig_Default(t *testing.T) {
	var c WaiterConfig
	c.Default()

	assertConfigDefaults(t, c)
}

func TestWaiter_NewWaiter(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		w := NewWaiter(nil, nil)

		assertConfigDefaults(t, w.config)
	})

	t.Run("custom options", func(t *testing.T) {
		interval := 7 * time.Second
		timeout := 8 * time.Second

		w := NewWaiter(nil, nil,
			WithInterval(interval),
			WithTimeout(timeout))

		assert.Equal(t, interval, w.config.Interval)
		assert.Equal(t, timeout, w.config.Timeout)
	})
}

func assertConfigDefaults(t *testing.T, c WaiterConfig) {
	t.Helper()
	assert.Equal(t, WaiterDefaultInterval, c.Interval)
	assert.Equal(t, WaiterDefaultTimeout, c.Timeout)
}

func Test_checkObjectCondition(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))

	tests := []struct {
		name   string
		object client.Object
		result bool
	}{
		{
			name: "structured deployment",
			object: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test",
					Namespace:  "test",
					Generation: 5,
				},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: 5,
					Conditions: []appsv1.DeploymentCondition{
						{
							Type:   appsv1.DeploymentAvailable,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			result: true,
		},
		{
			name: "outdated structured deployment",
			object: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test",
					Namespace:  "test",
					Generation: 5,
				},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: 1,
					Conditions: []appsv1.DeploymentCondition{
						{
							Type:   appsv1.DeploymentAvailable,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			result: false,
		},
		{
			name: "outdated unstructured",
			object: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"generation": int64(5),
					},
					"status": map[string]interface{}{
						"conditions": []map[string]interface{}{
							{
								"type":               "Available",
								"observedGeneration": 3,
							},
						},
					},
				},
			},
			result: false,
		},
		{
			name: "up-to-date unstructured",
			object: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"generation": int64(5),
					},
					"status": map[string]interface{}{
						"conditions": []map[string]interface{}{
							{
								"type":               "Available",
								"status":             "True",
								"observedGeneration": int64(5),
							},
						},
					},
				},
			},
			result: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			done, err := checkObjectCondition(test.object,
				"Available", metav1.ConditionTrue, scheme)
			require.NoError(t, err)
			assert.Equal(t, test.result, done)
		})
	}
}
