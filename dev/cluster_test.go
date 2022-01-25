package dev

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestClusterConfig_Default(t *testing.T) {
	c := ClusterConfig{WorkDir: "test"}
	c.Default()

	assertClusterConfigDefaults(t, c)
}

func assertClusterConfigDefaults(t *testing.T, c ClusterConfig) {
	t.Helper()
	assert.NotNil(t, c.Logger)
	// can't compare functions, so we just make sure something is defaulted
	assert.NotNil(t, c.NewWaiter)
	assert.NotNil(t, c.NewHelm)
	assert.NotNil(t, c.NewRestConfig)
	assert.NotNil(t, c.NewCtrlClient)

	assert.NotNil(t, c.Logger)
	assert.Equal(t, []WaitOption{WithLogger(c.Logger)}, c.WaitOptions)

	// Workdir is mandatory in NewCluster()
	assert.Equal(t, "test/kubeconfig.yaml", c.Kubeconfig)
}

func TestCluster_NewCluster(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		var newRestConfig NewRestConfigMock
		var newCtrlClient NewCtrlClientMock

		newRestConfig.
			On("New", mock.Anything).
			Return((*rest.Config)(nil), nil)

		newCtrlClient.
			On("New", mock.Anything, mock.Anything).
			Return(nil, nil)

		c, err := NewCluster("test",
			WithNewRestConfigFunc(newRestConfig.New),
			WithNewCtrlClientFunc(newCtrlClient.New))
		require.NoError(t, err)

		assertClusterConfigDefaults(t, c.config)
		assert.Equal(t, "test/kubeconfig.yaml", c.Kubeconfig())
	})
}

type NewRestConfigMock struct {
	mock.Mock
}

func (m *NewRestConfigMock) New(kubeconfig string) (*rest.Config, error) {
	args := m.Called(kubeconfig)
	return args.Get(0).(*rest.Config), args.Error(1)
}

type NewCtrlClientMock struct {
	mock.Mock
}

func (m *NewCtrlClientMock) New(
	c *rest.Config, opts client.Options) (client.Client, error) {
	args := m.Called(c, opts)
	// TODO: come up with a smart way to return a nil client:
	return nil, args.Error(1)
}
