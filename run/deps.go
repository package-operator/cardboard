package run

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/xlab/treeprint"
)

// Represents a dependency.
type Dependency interface {
	DependencyIDer
	// Executes the dependency.
	Run(ctx context.Context) error
}

// A depdencency that can uniquely identify itself.
type DependencyIDer interface {
	// Unique Identifier to ensure this dependency only executes once.
	ID() string
}

// Uses a string as dependency identifier.
type DependencyID string

func (id DependencyID) ID() string {
	return string(id)
}

// Returns a new DependencyRun context.
func newDependencyRun() *dependencyRun {
	return &dependencyRun{
		ran:    map[string]*depOnce{},
		childs: map[string][]string{},
	}
}

type dependencyRun struct {
	// remembers functions that have already been executed.
	ran    map[string]*depOnce
	root   string
	childs map[string][]string
	mux    sync.Mutex
}

func (r *dependencyRun) Report() string {
	var report bytes.Buffer
	fmt.Fprintln(&report, "Cardboard Report:")
	for _, child := range r.childs[r.root] {
		root := treeprint.NewWithRoot(r.printNode(child))
		r.traverseTree(root, r.childs[child])

		tree := strings.TrimSpace(root.String())
		if len(tree) > 0 {
			fmt.Fprintln(&report, tree)
		}
	}
	return report.String()
}

func (r *dependencyRun) traverseTree(t treeprint.Tree, childs []string) {
	for _, child := range childs {
		txt := r.printNode(child)
		r.traverseTree(t.AddBranch(txt), r.childs[child])
	}
}

const (
	greenColor  = "\033[32m"
	redColor    = "\033[31m"
	yellowColor = "\033[33m"
	resetColor  = "\033[0m"
)

func colorize(t, color string) string {
	if NoColor {
		return t
	}
	return color + t + resetColor
}

func (r *dependencyRun) printNode(child string) string {
	entry := r.ran[child]
	var txt string
	if entry.err == nil {
		txt += colorize("[OK] ", greenColor)
	} else {
		txt += colorize("[ERR] ", redColor)
	}
	txt += child
	txt += colorize(fmt.Sprintf(" [took %s]", entry.took), yellowColor)
	if entry.err != nil && !r.childHasError(r.childs[child]) {
		txt += "\n" + colorize(entry.err.Error(), redColor)
	}
	return txt
}

func (r *dependencyRun) childHasError(childs []string) bool {
	for _, child := range childs {
		entry := r.ran[child]
		if entry.err != nil {
			return true
		}
	}
	return false
}

// Executes dependencies in parallel.
func (r *dependencyRun) Parallel(ctx context.Context, parent DependencyIDer, deps ...Dependency) error {
	var (
		wg      sync.WaitGroup
		errs    []error
		errsMux sync.Mutex
	)

	wg.Add(len(deps))
	localDeps := make([]Dependency, len(deps))
	for i, dep := range deps {
		localDeps[i] = r.get(dep, parent.ID())
	}
	for _, dep := range localDeps {
		dep := dep
		go func() {
			defer wg.Done()
			if err := dep.Run(ctx); err != nil {
				errsMux.Lock()
				errs = append(errs, fmt.Errorf("running %s: %w", dep.ID(), err))
				errsMux.Unlock()
			}
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

// Executes dependencies one after the other.
func (r *dependencyRun) Serial(ctx context.Context, parent DependencyIDer, deps ...Dependency) error {
	localDeps := make([]Dependency, len(deps))
	for i, dep := range deps {
		localDeps[i] = r.get(dep, parent.ID())
	}
	for _, dep := range localDeps {
		if err := dep.Run(ctx); err != nil {
			return fmt.Errorf("running %s: %w", dep.ID(), err)
		}
	}
	return nil
}

func (r *dependencyRun) get(dep Dependency, parent string) Dependency {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.root == "" {
		r.root = parent
	}

	r.childs[parent] = append(r.childs[parent], dep.ID())
	out, ok := r.ran[dep.ID()]
	if !ok {
		out = newOnce(dep)
		r.ran[dep.ID()] = out
	}
	return out
}

type dep struct {
	id  string
	run func(ctx context.Context) error
}

func (d *dep) ID() string {
	return d.id
}

func (d *dep) Run(ctx context.Context) error {
	return d.run(ctx)
}

type fn interface {
	func() | func() error | func(ctx context.Context) | func(ctx context.Context) error
}

// Wraps a struct method with no parameters for the dependency handler.
func Meth[T fn](s any, fn T) Dependency {
	return FnWithName(methID(s, fn), fn)
}

// Wraps a function with no parameters for the dependency handler.
func Fn[T fn](fn T) Dependency {
	return FnWithName(funcID(fn), fn)
}

// Wraps a function with no parameters and a specific name for the dependency handler.
func FnWithName[T fn](name string, fn T) Dependency {
	return &dep{
		id: name,
		run: func(ctx context.Context) error {
			switch v := any(fn).(type) {
			case func():
				v()
			case func() error:
				return v()
			case func(context.Context):
				v(ctx)
			case func(context.Context) error:
				return v(ctx)
			}
			return nil
		},
	}
}

type fn1[A any] interface {
	func(A) | func(A) error | func(context.Context, A) | func(context.Context, A) error
}

// Wraps a struct method with one parameters for the dependency handler.
func Meth1[T fn1[A], A any](s any, fn T, a1 A) Dependency {
	return Fn1WithName(methID(s, fn, a1), fn, a1)
}

// Wraps a function with one parameter for the dependency handler.
func Fn1[T fn1[A], A any](fn T, a1 A) Dependency {
	return Fn1WithName(funcID(fn, a1), fn, a1)
}

// Wraps a function with one parameter and a specific name for the dependency handler.
func Fn1WithName[T fn1[A], A any](name string, fn T, a1 A) Dependency {
	return &dep{
		id: name,
		run: func(ctx context.Context) error {
			switch v := any(fn).(type) {
			case func(A):
				v(a1)
			case func(A) error:
				return v(a1)
			case func(context.Context, A):
				v(ctx, a1)
			case func(context.Context, A) error:
				return v(ctx, a1)
			}
			return nil
		},
	}
}

type fn2[A, B any] interface {
	func(A, B) | func(A, B) error | func(context.Context, A, B) | func(context.Context, A, B) error
}

// Wraps a struct method with two parameters for the dependency handler.
func Meth2[T fn2[A, B], A, B any](s any, fn T, a1 A, a2 B) Dependency {
	return Fn2WithName(methID(s, fn, a1, a2), fn, a1, a2)
}

// Wraps a function with two parameters for the dependency handler.
func Fn2[T fn2[A, B], A, B any](fn T, a1 A, a2 B) Dependency {
	return Fn2WithName(funcID(fn, a1, a2), fn, a1, a2)
}

// Wraps a function with two parameters and a specific name for the dependency handler.
func Fn2WithName[T fn2[A, B], A, B any](name string, fn T, a1 A, a2 B) Dependency {
	return &dep{
		id: name,
		run: func(ctx context.Context) error {
			switch v := any(fn).(type) {
			case func(A, B):
				v(a1, a2)
			case func(A, B) error:
				return v(a1, a2)
			case func(context.Context, A, B):
				v(ctx, a1, a2)
			case func(context.Context, A, B) error:
				return v(ctx, a1, a2)
			}
			return nil
		},
	}
}

type fn3[A, B, C any] interface {
	func(A, B, C) | func(A, B, C) error | func(context.Context, A, B, C) | func(context.Context, A, B, C) error
}

// Wraps a struct method with three parameters for the dependency handler.
func Meth3[T fn3[A, B, C], A, B, C any](s any, fn T, a1 A, a2 B, a3 C) Dependency {
	return Fn3WithName(methID(s, fn, a1, a2, a3), fn, a1, a2, a3)
}

// Wraps a function with three parameters for the dependency handler.
func Fn3[T fn3[A, B, C], A, B, C any](fn T, a1 A, a2 B, a3 C) Dependency {
	return Fn3WithName(funcID(fn, a1, a2, a3), fn, a1, a2, a3)
}

// Wraps a function with three parameters and a specific name for the dependency handler.
func Fn3WithName[T fn3[A, B, C], A, B, C any](name string, fn T, a1 A, a2 B, a3 C) Dependency {
	return &dep{
		id: name,
		run: func(ctx context.Context) error {
			switch v := any(fn).(type) {
			case func(A, B, C):
				v(a1, a2, a3)
			case func(A, B, C) error:
				return v(a1, a2, a3)
			case func(context.Context, A, B, C):
				v(ctx, a1, a2, a3)
			case func(context.Context, A, B, C) error:
				return v(ctx, a1, a2, a3)
			}
			return nil
		},
	}
}

type fn4[A, B, C, D any] interface {
	func(A, B, C, D) | func(A, B, C, D) error | func(context.Context, A, B, C, D) | func(context.Context, A, B, C, D) error
}

// Wraps a struct method with four parameters for the dependency handler.
func Meth4[T fn4[A, B, C, D], A, B, C, D any](s any, fn T, a1 A, a2 B, a3 C, a4 D) Dependency {
	return Fn4WithName(methID(s, fn, a1, a2, a3, a4), fn, a1, a2, a3, a4)
}

// Wraps a function with four parameters for the dependency handler.
func Fn4[T fn4[A, B, C, D], A, B, C, D any](fn T, a1 A, a2 B, a3 C, a4 D) Dependency {
	return Fn4WithName(funcID(fn, a1, a2, a3, a4), fn, a1, a2, a3, a4)
}

// Wraps a function with four parameters and a specific name for the dependency handler.
func Fn4WithName[T fn4[A, B, C, D], A, B, C, D any](name string, fn T, a1 A, a2 B, a3 C, a4 D) Dependency {
	return &dep{
		id: name,
		run: func(ctx context.Context) error {
			switch v := any(fn).(type) {
			case func(A, B, C, D):
				v(a1, a2, a3, a4)
			case func(A, B, C, D) error:
				return v(a1, a2, a3, a4)
			case func(context.Context, A, B, C, D):
				v(ctx, a1, a2, a3, a4)
			case func(context.Context, A, B, C, D) error:
				return v(ctx, a1, a2, a3, a4)
			}
			return nil
		},
	}
}

type fn5[A, B, C, D, E any] interface {
	func(A, B, C, D, E) | func(A, B, C, D, E) error | func(context.Context, A, B, C, D, E) | func(context.Context, A, B, C, D, E) error
}

// Wraps a struct method with five parameters for the dependency handler.
func Meth5[T fn5[A, B, C, D, E], A, B, C, D, E any](s any, fn T, a1 A, a2 B, a3 C, a4 D, a5 E) Dependency {
	return Fn5WithName(methID(s, fn, a1, a2, a3, a4, a5), fn, a1, a2, a3, a4, a5)
}

// Wraps a function with five parameters for the dependency handler.
func Fn5[T fn5[A, B, C, D, E], A, B, C, D, E any](fn T, a1 A, a2 B, a3 C, a4 D, a5 E) Dependency {
	return Fn5WithName(funcID(fn, a1, a2, a3, a4, a5), fn, a1, a2, a3, a4, a5)
}

// Wraps a function with five parameters and a specific name for the dependency handler.
func Fn5WithName[T fn5[A, B, C, D, E], A, B, C, D, E any](name string, fn T, a1 A, a2 B, a3 C, a4 D, a5 E) Dependency {
	return &dep{
		id: name,
		run: func(ctx context.Context) error {
			switch v := any(fn).(type) {
			case func(A, B, C, D, E):
				v(a1, a2, a3, a4, a5)
			case func(A, B, C, D, E) error:
				return v(a1, a2, a3, a4, a5)
			case func(context.Context, A, B, C, D, E):
				v(ctx, a1, a2, a3, a4, a5)
			case func(context.Context, A, B, C, D, E) error:
				return v(ctx, a1, a2, a3, a4, a5)
			}
			return nil
		},
	}
}

type fn6[A, B, C, D, E, F any] interface {
	func(A, B, C, D, E, F) | func(A, B, C, D, E, F) error | func(context.Context, A, B, C, D, E, F) | func(context.Context, A, B, C, D, E, F) error
}

// Wraps a struct method with six parameters for the dependency handler.
func Meth6[T fn6[A, B, C, D, E, F], A, B, C, D, E, F any](s any, fn T, a1 A, a2 B, a3 C, a4 D, a5 E, a6 F) Dependency {
	return Fn6WithName(methID(s, fn, a1, a2, a3, a4, a5, a6), fn, a1, a2, a3, a4, a5, a6)
}

// Wraps a function with six parameters for the dependency handler.
func Fn6[T fn6[A, B, C, D, E, F], A, B, C, D, E, F any](fn T, a1 A, a2 B, a3 C, a4 D, a5 E, a6 F) Dependency {
	return Fn6WithName(funcID(fn, a1, a2, a3, a4, a5, a6), fn, a1, a2, a3, a4, a5, a6)
}

// Wraps a function with six parameters and a specific name for the dependency handler.
func Fn6WithName[T fn6[A, B, C, D, E, F], A, B, C, D, E, F any](name string, fn T, a1 A, a2 B, a3 C, a4 D, a5 E, a6 F) Dependency {
	return &dep{
		id: name,
		run: func(ctx context.Context) error {
			switch v := any(fn).(type) {
			case func(A, B, C, D, E, F):
				v(a1, a2, a3, a4, a5, a6)
			case func(A, B, C, D, E, F) error:
				return v(a1, a2, a3, a4, a5, a6)
			case func(context.Context, A, B, C, D, E, F):
				v(ctx, a1, a2, a3, a4, a5, a6)
			case func(context.Context, A, B, C, D, E, F) error:
				return v(ctx, a1, a2, a3, a4, a5, a6)
			}
			return nil
		},
	}
}

type selfIdentifier interface {
	ID() string
}

func methID(thing, fn any, args ...any) string {
	sid := structID(thing)
	fid := funcID(fn, args...)
	idx := strings.LastIndex(fid, ").")
	if idx >= 0 {
		// if the function receiver is a pointer, fid will look like "main.(*CI).PreCommit([]string{})"
		// we want to remove the "main.(*CI)" part to replace it with sid
		fid = fid[idx+2:]
	} else {
		sidSlice := sid
		sIdx := strings.Index(sid, "{")
		if sIdx >= 0 {
			sidSlice = sid[:sIdx]
		}
		idx = strings.LastIndex(fid, sidSlice)
		if idx >= 0 {
			// if the function receiver is not a pointer, fid will look like "main.Lint.glciFix()"
			// we want to remove the "main.Lint" part to replace it with sid
			fid = fid[(idx + len(sidSlice) + 1):]
		}
	}
	return fmt.Sprintf("%s.%s", sid, fid)
}

func methIDLit(thing any, fn string, args ...any) string {
	sid := structID(thing)
	if len(args) > 0 {
		argStrings := make([]string, len(args))
		for i, arg := range args {
			argStrings[i] = fmt.Sprintf("%#v", arg)
		}
		return fmt.Sprintf("%s.%s(%s)", sid, fn, strings.Join(argStrings, ", "))
	}
	return fmt.Sprintf("%s.%s()", sid, fn)
}

// returns a string to identify a struct.
func structID(thing any) string {
	if sid, ok := thing.(selfIdentifier); ok {
		return sid.ID()
	}

	thingType := reflect.TypeOf(thing)
	if thingType.Kind() == reflect.Pointer {
		thingType = thingType.Elem()
	}
	if thingType.Kind() != reflect.Struct {
		// TODO: error?
		return ""
	}
	fields := strings.TrimPrefix(fmt.Sprintf("%+v", thing), "&")
	return fmt.Sprintf("%s.%s%s", thingType.PkgPath(), thingType.Name(), fields)
}

// returns a string that can be used to identify the given function and arguments.
func funcID(fn any, args ...any) string {
	fnV := reflect.ValueOf(fn)
	fnR := runtime.FuncForPC(fnV.Pointer())
	name := strings.TrimSuffix(fnR.Name(), "-fm")
	if len(args) > 0 {
		argStrings := make([]string, len(args))
		for i, arg := range args {
			argStrings[i] = fmt.Sprintf("%#v", arg)
		}
		return fmt.Sprintf("%s(%s)", name, strings.Join(argStrings, ", "))
	}
	return name + "()"
}

// container type to ensure a dependency only runs once.
type depOnce struct {
	once *sync.Once
	dep  Dependency
	took time.Duration
	err  error
}

func newOnce(dep Dependency) *depOnce {
	return &depOnce{
		once: &sync.Once{},
		dep:  dep,
	}
}

func (o *depOnce) ID() string {
	return o.dep.ID()
}

func (o *depOnce) Run(ctx context.Context) error {
	o.once.Do(func() {
		start := time.Now()
		defer func() {
			a := recover()
			if a == nil {
				return
			}

			var mustErr *MustError
			err, ok := a.(error)
			if ok && errors.As(err, &mustErr) {
				o.err = mustErr
				return
			}
			o.err = &internalPanickedError{
				Obj:   a,
				Stack: string(debug.Stack()),
			}
		}()

		o.err = o.dep.Run(ctx)
		o.took = time.Since(start)
	})
	return o.err
}
