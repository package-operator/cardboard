package sh_test

import (
	"testing"

	"github.com/neilotoole/slogt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pkg.package-operator.run/cardboard/sh"
)

func TestRunner_Run(t *testing.T) {
	t.Parallel()
	log := slogt.New(t)
	err := sh.New(sh.WithLogger{log}).Run(t.Context(), "echo")
	require.NoError(t, err)
}

func TestRunner_Run_error(t *testing.T) {
	t.Parallel()
	log := slogt.New(t)
	err := sh.New(sh.WithLogger{log}).Run(t.Context(), "bash", "-c", "false")
	require.EqualError(t, err, "running \"bash -c false\" failed with exit code 1")
}

func TestRunner_Run_runerror(t *testing.T) {
	t.Parallel()
	log := slogt.New(t)
	err := sh.New(sh.WithLogger{log}).Run(t.Context(), "xxxxxxxxxxx")
	require.EqualError(t, err, "failed to run \"xxxxxxxxxxx \": exec: \"xxxxxxxxxxx\": executable file not found in $PATH")
}

func TestRunner_Output(t *testing.T) {
	t.Parallel()
	log := slogt.New(t)
	out, err := sh.New(sh.WithLogger{log}).Output(t.Context(), "echo", "hello world")
	require.NoError(t, err)
	assert.Equal(t, "hello world", out)
}
