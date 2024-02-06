package run

import (
	"context"
	"embed"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"os"
	"reflect"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/mattn/go-isatty"
)

type ManagerOption interface {
	ApplyToManager(m *Manager)
}

// Provide a custom logger to the Manager.
type WithLogger struct{ *slog.Logger }

func (l WithLogger) ApplyToManager(m *Manager) {
	m.logger = l.Logger
}

// Add dependencies that will be run in parallel. Parallel dependencies are run before Serial dependencies.
type WithParallelDeps []Dependency

func (pd WithParallelDeps) ApplyToManager(m *Manager) {
	m.parallel = pd
}

// Add dependencies that will be run in series. Serial dependencies run after Parallel dependencies.
type WithSerialDeps []Dependency

func (pd WithSerialDeps) ApplyToManager(m *Manager) {
	m.serial = pd
}

// Source code to use for Help generation.
// Allows the automatic detection of method comments.
// Example go-embed directive:
// //go:embed *.go
// var source embed.FS.
type WithSources embed.FS

func (s WithSources) ApplyToManager(m *Manager) {
	m.sources = embed.FS(s)
}

type WithStdout struct{ io.Writer }

func (stdout WithStdout) ApplyToManager(m *Manager) {
	m.stdout = stdout.Writer
}

type WithStderr struct{ io.Writer }

func (stderr WithStderr) ApplyToManager(m *Manager) {
	m.stderr = stderr.Writer
}

var NoColor = os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" ||
	(!isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()))

// Manager coordinates runnable targets and dependencies.
type Manager struct {
	logger         *slog.Logger
	helpTargets    []string
	targets        map[string]target
	runOnce        sync.Once
	dr             *dependencyRun
	dm             *dependencyManager
	stdout, stderr io.Writer

	// config
	sources  embed.FS
	parallel []Dependency
	serial   []Dependency
}

type target struct {
	idWithArgs func(args ...any) string
	run        func(ctx context.Context, args []string) error
}

// Creates a new Manager.
func New(opts ...ManagerOption) *Manager {
	dr := newDependencyRun()
	m := &Manager{
		targets: map[string]target{},
		dr:      dr,
		dm:      newDependencyManager(dr),
	}
	for _, opt := range opts {
		opt.ApplyToManager(m)
	}
	if m.logger == nil {
		m.logger = slog.Default()
	}
	if m.stderr == nil {
		m.stderr = os.Stderr
	}
	if m.stdout == nil {
		m.stdout = os.Stdout
	}
	return m
}

// Executes dependencies one after the other.
func (m *Manager) SerialDeps(ctx context.Context, parent DependencyIDer, deps ...Dependency) error {
	return m.dr.Serial(ctx, parent, deps...)
}

// Executes dependencies in parallel.
func (m *Manager) ParallelDeps(ctx context.Context, parent DependencyIDer, deps ...Dependency) error {
	return m.dr.Parallel(ctx, parent, deps...)
}

func (m *Manager) Register(things ...any) error {
	return decorateWithCallingSourceLine(
		m.registerAll(things...),
	)
}

func (m *Manager) Call(ctx context.Context, id string, args []string) (err error) {
	target, ok := m.targets[id]
	if !ok {
		return &UnknownTargetError{ID: id}
	}
	return m.dr.Serial(ctx, DependencyID("."), FnWithName(target.idWithArgs(args), func() error {
		return target.run(ctx, args)
	}))
}

func (m *Manager) Run(ctx context.Context) error {
	var err error
	m.runOnce.Do(func() {
		// Make sure deps are in the path for everything we run.
		os.Setenv("PATH", m.dm.Bin()+":"+os.Getenv("PATH"))

		err = m.run(ctx)
	})
	return err
}

// Register a go tool to be installed.
// The manager ensures that the tool is go install'ed project local and available in $PATH.
func (m *Manager) RegisterGoTool(tool, packageURL, version string) error {
	return m.dm.Register(tool, packageURL, version)
}

func (m *Manager) RegisterAndRun(ctx context.Context, things ...any) error {
	if err := m.registerAll(things...); err != nil {
		return decorateWithCallingSourceLine(err)
	}
	if err := m.Run(ctx); err != nil {
		return err
	}

	return nil
}

func (m *Manager) MustRegisterAndRun(ctx context.Context, things ...any) {
	if err := m.registerAll(things...); err != nil {
		m.logger.Error((decorateWithCallingSourceLine(err)).Error())
		os.Exit(1)
	}
	if err := m.Run(ctx); err != nil {
		m.logger.Error(err.Error())
		os.Exit(1)
	}
}

func (m *Manager) printHelp() error {
	docs := map[string]string{}
	docPkg, err := commentsFromSource(m.sources)
	if err != nil {
		return err
	}
	for _, t := range docPkg.Types {
		docs[t.Name] = strings.TrimSpace(t.Doc)
		for _, m := range t.Methods {
			docs[t.Name+":"+m.Name] = strings.TrimSpace(m.Doc)
		}
	}

	namespaceIndex := map[string]struct{}{}
	methods := map[string][]string{}
	for _, t := range m.helpTargets {
		parts := strings.Split(t, ":")
		ns := parts[0]
		meth := parts[1]
		namespaceIndex[ns] = struct{}{}
		methods[ns] = append(methods[ns], meth)
	}
	namespaces := make([]string, len(namespaceIndex))
	var i int
	for k := range namespaceIndex {
		namespaces[i] = k
		i++
	}
	slices.Sort(namespaces)
	for k := range methods {
		slices.Sort(methods[k])
	}

	fmt.Fprintln(m.stdout, "Autogenerated help, available targets:")
	w := tabwriter.NewWriter(m.stdout, 0, 0, 2, ' ', tabwriter.TabIndent)
	for _, ns := range namespaces {
		if len(docs[ns]) > 0 {
			fmt.Fprintf(w, "\n%s\t%s\n", ns, docs[ns])
		} else {
			fmt.Fprintf(w, "\n%s\n", ns)
		}
		for _, meth := range methods[ns] {
			fn := ns + ":" + meth
			if len(docs[fn]) > 0 {
				fmt.Fprintf(w, "- %s\t%s\n", fn, docs[fn])
				continue
			}
			fmt.Fprintf(w, "- %s\n", fn)
		}
	}
	return w.Flush()
}

func (m *Manager) run(ctx context.Context) error {
	args := os.Args
	if len(args) < 2 || args[1] == "help" {
		return m.printHelp()
	}

	// Always do binary dependencies first.
	if !m.dm.IsEmpty() {
		if err := m.dr.Serial(ctx, DependencyID("."), m.dm); err != nil {
			return err
		}
	}

	// All other parallel deps.
	if err := m.dr.Parallel(ctx, DependencyID("."), m.parallel...); err != nil {
		return fmt.Errorf("parallel dependency failed: %w", err)
	}
	// All other serial deps.
	if err := m.dr.Serial(ctx, DependencyID("."), m.serial...); err != nil {
		return fmt.Errorf("serial dependency failed: %w", err)
	}

	// Execute actual target.
	err := m.Call(ctx, args[1], args[2:])
	fmt.Fprint(m.stdout, m.dr.Report())
	return err
}

func (m *Manager) registerAll(things ...any) error {
	for _, thing := range things {
		if err := m.register(thing); err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}
	}
	return nil
}

func (m *Manager) register(thing any) error {
	if thing == nil {
		return fmt.Errorf("thing must not be nil")
	}

	thingType := reflect.TypeOf(thing)
	if thingType.Kind() != reflect.Pointer ||
		thingType.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("thing must be pointer to struct")
	}

	thingValue := reflect.ValueOf(thing)
	typeID := thingType.Elem().Name()
	for i := 0; i < thingType.NumMethod(); i++ {
		method := thingType.Method(i)
		methodValue := thingValue.MethodByName(method.Name)
		if err := m.registerMeth(typeID, thing, method, methodValue); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) registerMeth(
	typeID string, thing any,
	method reflect.Method,
	methodValue reflect.Value,
) error {
	if !method.IsExported() {
		return nil
	}
	if method.Name == "ID" && method.Type.NumIn() == 1 && method.Type.NumOut() == 1 {
		return nil
	}

	methodID := method.Name
	// check for name collision
	targetID := typeID + ":" + methodID
	lowerTargetID := strings.ToLower(targetID)

	_, exists := m.targets[targetID]
	_, lowerExists := m.targets[lowerTargetID]
	if exists || lowerExists {
		return fmt.Errorf("a target for %s is already registered", targetID)
	}

	// check params
	if method.Type.NumIn() != 3 || method.Type.NumOut() != 1 ||
		!(method.Type.In(1).String() == "context.Context") ||
		!(method.Type.In(2).String() == "[]string") ||
		!(method.Type.Out(0).String() == "error") {
		return fmt.Errorf(
			"%s.%s() must have signature like func(context.Context, []string) error",
			typeID, method.Name)
	}

	fn := func(ctx context.Context, args []string) (err error) {
		defer func() {
			a := recover()
			if a == nil {
				return
			}
			var ok bool
			if err, ok = a.(error); ok {
				return
			}

			err = &internalPanickedError{
				ID:    typeID + ":" + methodID,
				Obj:   a,
				Stack: string(debug.Stack()),
			}
		}()

		out := methodValue.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(args),
		})
		errI := out[0].Interface()
		if errI == nil {
			return nil
		}
		return errI.(error)
	}
	t := target{
		idWithArgs: func(args ...any) string {
			return methIDLit(thing, methodID, args...)
		},
		run: fn,
	}
	m.targets[targetID] = t
	m.targets[lowerTargetID] = t
	m.helpTargets = append(m.helpTargets, targetID)
	return nil
}

func commentsFromSource(source embed.FS) (*doc.Package, error) {
	fileSet := token.NewFileSet()
	files := map[string]*ast.File{}

	entries, err := source.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := source.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read embedded source: %w", err)
		}

		astFile, err := parser.ParseFile(fileSet, entry.Name(), data, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse AST: %w", err)
		}
		files[entry.Name()] = astFile
	}

	// This will fail due to unresolved imports,
	// but we don't care for just generating documentation.
	apkg, _ := ast.NewPackage(fileSet, files, nil, nil)
	return doc.New(apkg, "", 0), nil
}
