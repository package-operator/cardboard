package run

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/neilotoole/slogt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MyThing for unittesting.
type MyThing struct {
	field string
	mgr   *Manager
}

func (m *MyThing) ID() string {
	// omit mgr field from id.
	return fmt.Sprintf("pkg.package-operator.run/cardboard/run.MyThing{field:%s}", m.field)
}

// Docs for Test123.
func (m *MyThing) Test123(_ context.Context, _ []string) error {
	return nil
}

func (m *MyThing) TestPanic(_ context.Context, _ []string) error {
	panic("xxx")
}

// Docs for ValueReceiver.
func (m MyThing) ValueReceiver(_ context.Context, _ []string) error {
	return nil
}

// Docs for TestWithDep.
func (m *MyThing) TestWithDep(ctx context.Context, args []string) error {
	return m.mgr.SerialDeps(ctx,
		Meth1(m, m.TestWithDep, args),
		Meth1(m, m.private, "banana"))
}

// Docs for TestWithDepErr.
func (m *MyThing) TestWithDepErr(ctx context.Context, args []string) error {
	return m.mgr.SerialDeps(ctx,
		Meth1(m, m.TestWithDepErr, args),
		Meth1(m, m.privateErr, "banana"))
}

// Docs for TestWithDepMustPanic.
func (m *MyThing) TestWithDepMustPanic(ctx context.Context, args []string) error {
	Must(m.mgr.SerialDeps(ctx,
		Meth1(m, m.TestWithDepMustPanic, args),
		Meth1(m, m.privateMustPanic, "banana")))
	return nil
}

func (m *MyThing) private(_ string) {}
func (m *MyThing) privateErr(_ string) error {
	return errors.New("explosion")
}

func (m *MyThing) privateMustPanic(_ string) {
	Must(errors.New("explosion"))
}

func (m MyThing) privateReceiverNotPointer(_ string) {
}

func TestManager_Call_unknownTarget(t *testing.T) {
	log := slogt.New(t)
	mgr := New(WithLogger{log})

	ctx := t.Context()
	err := mgr.Call(ctx, "Test:Banana", []string{})
	require.EqualError(t, err, `unknown target: "Test:Banana"`)
	var unknownTargetErr *UnknownTargetError
	require.ErrorAs(t, err, &unknownTargetErr)
}

func TestManager_Call(t *testing.T) {
	log := slogt.New(t)
	mgr := New(WithLogger{log})
	require.NoError(t, mgr.Register(&MyThing{}))

	ctx := t.Context()
	err := mgr.Call(ctx, "MyThing:Test123", []string{})
	require.NoError(t, err)
}

func TestManager_Call_panics(t *testing.T) {
	log := slogt.New(t)
	mgr := New(WithLogger{log})
	require.NoError(t, mgr.Register(&MyThing{}))

	ctx := t.Context()
	err := mgr.Call(ctx, "MyThing:TestPanic", []string{})
	require.Error(t, err)
}

func TestManager_Run(t *testing.T) {
	log := slogt.New(t)
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)
	mgr := New(WithLogger{log}, WithStderr{&stderrBuf}, WithStdout{&stdoutBuf})
	require.NoError(t, mgr.Register(&MyThing{field: "hans", mgr: mgr}))

	os.Args = []string{"", "MyThing:TestWithDep"}
	ctx := t.Context()
	require.NoError(t, mgr.Run(ctx))

	assert.Empty(t, stdoutBuf.String())
	// strip [took x] from output, because it is not stable.
	tookRegEx := regexp.MustCompile(`(?m) \[took .*\]`)
	assert.Equal(t, `Cardboard Report:
[OK] pkg.package-operator.run/cardboard/run.MyThing{field:hans}.TestWithDep([]string{})
└── [OK] pkg.package-operator.run/cardboard/run.MyThing{field:hans}.private("banana")
`, string(tookRegEx.ReplaceAll(stderrBuf.Bytes(), nil)))
}

func TestManager_Run_Error(t *testing.T) {
	log := slogt.New(t)
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)
	mgr := New(WithLogger{log}, WithStderr{&stderrBuf}, WithStdout{&stdoutBuf})
	require.NoError(t, mgr.Register(&MyThing{field: "hans", mgr: mgr}))

	os.Args = []string{"", "MyThing:TestWithDepErr"}
	ctx := t.Context()
	require.EqualError(t, mgr.Run(ctx),
		`running pkg.package-operator.run/cardboard/run.MyThing{field:hans}.TestWithDepErr([]string{}): `+
			`running pkg.package-operator.run/cardboard/run.MyThing{field:hans}.privateErr("banana"): explosion`)

	assert.Empty(t, stdoutBuf.String())
	// strip [took x] from output, because it is not stable.
	tookRegEx := regexp.MustCompile(`(?m) \[took .*\]`)
	assert.Equal(t, `Cardboard Report:
[ERR] pkg.package-operator.run/cardboard/run.MyThing{field:hans}.TestWithDepErr([]string{})
└── [ERR] pkg.package-operator.run/cardboard/run.MyThing{field:hans}.privateErr("banana")
    explosion
`, string(tookRegEx.ReplaceAll(stderrBuf.Bytes(), nil)))
}

func TestManager_Run_MustPanic(t *testing.T) {
	log := slogt.New(t)
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)
	mgr := New(WithLogger{log}, WithStderr{&stderrBuf}, WithStdout{&stdoutBuf})
	require.NoError(t, mgr.Register(&MyThing{field: "hans", mgr: mgr}))

	os.Args = []string{"", "MyThing:TestWithDepMustPanic"}
	ctx := t.Context()
	require.EqualError(t, mgr.Run(ctx),
		`running pkg.package-operator.run/cardboard/run.MyThing{field:hans}.TestWithDepMustPanic([]string{}): `+
			`running pkg.package-operator.run/cardboard/run.MyThing{field:hans}.privateMustPanic("banana"): explosion`)

	assert.Empty(t, stdoutBuf.String())
	// strip [took x] from output, because it is not stable.
	tookRegEx := regexp.MustCompile(`(?m) \[took .*\]`)
	assert.Equal(t, `Cardboard Report:
[ERR] pkg.package-operator.run/cardboard/run.MyThing{field:hans}.TestWithDepMustPanic([]string{})
└── [ERR] pkg.package-operator.run/cardboard/run.MyThing{field:hans}.privateMustPanic("banana")
    explosion
`, string(tookRegEx.ReplaceAll(stderrBuf.Bytes(), nil)))
}

func TestManager_Run_help_withoutSource(t *testing.T) {
	log := slogt.New(t)
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)
	mgr := New(WithLogger{log}, WithStderr{&stderrBuf}, WithStdout{&stdoutBuf})
	require.NoError(t, mgr.Register(&MyThing{field: "hans", mgr: mgr}))

	os.Args = []string{"", "help"}
	ctx := t.Context()
	require.NoError(t, mgr.Run(ctx))

	assert.Empty(t, stderrBuf.String())
	// strip [took x] from output, because it is not stable.
	tookRegEx := regexp.MustCompile(`(?m) \[took .*\]`)
	assert.Equal(t, `Autogenerated help, available targets:

MyThing
- MyThing:Test123
- MyThing:TestPanic
- MyThing:TestWithDep
- MyThing:TestWithDepErr
- MyThing:TestWithDepMustPanic
- MyThing:ValueReceiver
`, string(tookRegEx.ReplaceAll(stdoutBuf.Bytes(), nil)))
}

//go:embed *.go
var source embed.FS

func TestManager_Run_help_withSource(t *testing.T) {
	log := slogt.New(t)
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)
	mgr := New(WithLogger{log}, WithStderr{&stderrBuf}, WithStdout{&stdoutBuf}, WithSources(source))
	require.NoError(t, mgr.Register(&MyThing{field: "hans", mgr: mgr}))

	os.Args = []string{"", "help"}
	ctx := t.Context()
	require.NoError(t, mgr.Run(ctx))

	assert.Empty(t, stderrBuf.String())
	// strip [took x] from output, because it is not stable.
	tookRegEx := regexp.MustCompile(`(?m) \[took .*\]`)
	assert.Equal(t, `Autogenerated help, available targets:

MyThing
- MyThing:Test123
- MyThing:TestPanic
- MyThing:TestWithDep
- MyThing:TestWithDepErr
- MyThing:TestWithDepMustPanic
- MyThing:ValueReceiver
`, string(tookRegEx.ReplaceAll(stdoutBuf.Bytes(), nil)))
}
