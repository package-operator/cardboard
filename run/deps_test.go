package run

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_newDependencyRun_Serial(t *testing.T) {
	t.Parallel()

	t.Run("returns error", func(t *testing.T) {
		t.Parallel()
		dr := newDependencyRun()
		ctx := context.Background()
		err := dr.Serial(ctx,
			DependencyID("_test"),
			Fn(func() error { return errTest }),
			Fn(func() error { return errTest }),
		)
		require.EqualError(t, err, "running pkg.package-operator.run/cardboard/run.Test_newDependencyRun_Serial.func1.1(): banana")
	})

	t.Run("simple", func(t *testing.T) {
		t.Parallel()
		dr := newDependencyRun()
		m := MyTestType{}
		ctx := context.Background()
		err := dr.Serial(ctx,
			DependencyID("_test"),
			Fn(myFunc),
			Fn1(m.Test1, "a"),
			Fn2(m.Test2, "a", "b"),
			Fn3(m.Test3, "a", 42, false),
			Fn4(m.Test4, "a", "b", "c", "d"),
			Fn5(m.Test5, "a", "b", "c", "d", "e"),
			Fn6(m.Test6, "a", "b", "c", "d", "e", "f"),
		)
		require.NoError(t, err)
	})

	t.Run("simple err", func(t *testing.T) {
		t.Parallel()
		dr := newDependencyRun()
		m := MyTestTypeErr{}
		ctx := context.Background()
		err := dr.Serial(ctx,
			DependencyID("_test"),
			Fn(myFuncErr),
			Fn1(m.Test1, "a"),
			Fn2(m.Test2, "a", "b"),
			Fn3(m.Test3, "a", 42, false),
			Fn4(m.Test4, "a", "b", "c", "d"),
			Fn5(m.Test5, "a", "b", "c", "d", "e"),
			Fn6(m.Test6, "a", "b", "c", "d", "e", "f"),
		)
		require.NoError(t, err)
	})

	t.Run("context err", func(t *testing.T) {
		t.Parallel()
		dr := newDependencyRun()
		m := MyTestTypeCtxErr{}
		ctx := context.Background()
		err := dr.Serial(ctx,
			DependencyID("_test"),
			Fn(myFuncCtxErr),
			Fn1(m.Test1, "a"),
			Fn2(m.Test2, "a", "b"),
			Fn3(m.Test3, "a", 42, false),
			Fn4(m.Test4, "a", "b", "c", "d"),
			Fn5(m.Test5, "a", "b", "c", "d", "e"),
			Fn6(m.Test6, "a", "b", "c", "d", "e", "f"),
		)
		require.NoError(t, err)
	})

	t.Run("context", func(t *testing.T) {
		t.Parallel()
		dr := newDependencyRun()
		m := MyTestTypeCtx{}
		ctx := context.Background()
		err := dr.Serial(ctx,
			DependencyID("_test"),
			Fn(myFuncCtx),
			Fn1(m.Test1, "a"),
			Fn2(m.Test2, "a", "b"),
			Fn3(m.Test3, "a", 42, false),
			Fn4(m.Test4, "a", "b", "c", "d"),
			Fn5(m.Test5, "a", "b", "c", "d", "e"),
			Fn6(m.Test6, "a", "b", "c", "d", "e", "f"),
		)
		require.NoError(t, err)
	})
}

var errTest = errors.New("banana")

type unwrapErrors interface {
	Unwrap() []error
}

func Test_newDependencyRun_Parallel(t *testing.T) {
	t.Parallel()
	t.Run("returns errors", func(t *testing.T) {
		t.Parallel()
		dr := newDependencyRun()
		ctx := context.Background()
		err := dr.Parallel(ctx,
			DependencyID("_test"),
			Fn(func() error { return errTest }),
			Fn(func() error { return errTest }),
		)

		//nolint:errorlint
		el := err.(unwrapErrors).Unwrap()
		require.Len(t, el, 2)

		errStrings := make([]string, len(el))
		for i, e := range el {
			errStrings[i] = e.Error()
		}
		assert.Contains(t, errStrings,
			"running pkg.package-operator.run/cardboard/run.Test_newDependencyRun_Parallel.func1.1(): banana")
		assert.Contains(t, errStrings,
			"running pkg.package-operator.run/cardboard/run.Test_newDependencyRun_Parallel.func1.2(): banana")
	})

	t.Run("simple", func(t *testing.T) {
		t.Parallel()
		dr := newDependencyRun()
		m := MyTestType{}
		ctx := context.Background()
		err := dr.Parallel(ctx,
			DependencyID("_test"),
			Fn(myFunc),
			Fn1(m.Test1, "a"),
			Fn2(m.Test2, "a", "b"),
			Fn3(m.Test3, "a", 42, false),
			Fn4(m.Test4, "a", "b", "c", "d"),
			Fn5(m.Test5, "a", "b", "c", "d", "e"),
			Fn6(m.Test6, "a", "b", "c", "d", "e", "f"),
		)
		require.NoError(t, err)
	})

	t.Run("simple err", func(t *testing.T) {
		t.Parallel()
		dr := newDependencyRun()
		m := MyTestTypeErr{}
		ctx := context.Background()
		err := dr.Parallel(ctx,
			DependencyID("_test"),
			Fn(myFuncErr),
			Fn1(m.Test1, "a"),
			Fn2(m.Test2, "a", "b"),
			Fn3(m.Test3, "a", 42, false),
			Fn4(m.Test4, "a", "b", "c", "d"),
			Fn5(m.Test5, "a", "b", "c", "d", "e"),
			Fn6(m.Test6, "a", "b", "c", "d", "e", "f"),
		)
		require.NoError(t, err)
	})

	t.Run("context err", func(t *testing.T) {
		t.Parallel()
		dr := newDependencyRun()
		m := MyTestTypeCtxErr{}
		ctx := context.Background()
		err := dr.Parallel(ctx,
			DependencyID("_test"),
			Fn(myFuncCtxErr),
			Fn1(m.Test1, "a"),
			Fn2(m.Test2, "a", "b"),
			Fn3(m.Test3, "a", 42, false),
			Fn4(m.Test4, "a", "b", "c", "d"),
			Fn5(m.Test5, "a", "b", "c", "d", "e"),
			Fn6(m.Test6, "a", "b", "c", "d", "e", "f"),
		)
		require.NoError(t, err)
	})

	t.Run("context", func(t *testing.T) {
		t.Parallel()
		dr := newDependencyRun()
		m := MyTestTypeCtx{}
		ctx := context.Background()
		err := dr.Parallel(ctx,
			DependencyID("_test"),
			Fn(myFuncCtx),
			Fn1(m.Test1, "a"),
			Fn2(m.Test2, "a", "b"),
			Fn3(m.Test3, "a", 42, false),
			Fn4(m.Test4, "a", "b", "c", "d"),
			Fn5(m.Test5, "a", "b", "c", "d", "e"),
			Fn6(m.Test6, "a", "b", "c", "d", "e", "f"),
		)
		require.NoError(t, err)
	})

	t.Run("context func", func(t *testing.T) {
		t.Parallel()
		dr := newDependencyRun()
		ctx := context.Background()
		err := dr.Parallel(ctx,
			Fn1(test1, "a"),
			Fn(myFuncCtx),
			Fn2(test2, "a", "b"),
			Fn3(test3, "a", 42, false),
			Fn4(test4, "a", "b", "c", "d"),
			Fn5(test5, "a", "b", "c", "d", "e"),
			Fn6(test6, "a", "b", "c", "d", "e", "f"),
		)
		require.NoError(t, err)
	})
}

func Test_funcID(t *testing.T) {
	t.Parallel()
	m := &MyTestType{}
	n := MyTestType{}
	tests := []struct {
		name     string
		fn       any
		args     []any
		expected string
	}{
		{
			name:     "closure",
			fn:       func(test string) {},
			args:     []any{"test"},
			expected: `pkg.package-operator.run/cardboard/run.Test_funcID.func1("test")`,
		},
		{
			name:     "pointer method",
			fn:       m.Test3,
			args:     []any{"1", 42, true},
			expected: `pkg.package-operator.run/cardboard/run.(*MyTestType).Test3("1", 42, true)`,
		},
		{
			name:     "value method",
			fn:       n.Test3,
			args:     []any{"1", 42, true},
			expected: `pkg.package-operator.run/cardboard/run.(*MyTestType).Test3("1", 42, true)`,
		},
		{
			name:     "function",
			fn:       myFunc,
			expected: `pkg.package-operator.run/cardboard/run.myFunc()`,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			out := funcID(test.fn, test.args...)
			assert.Equal(t, test.expected, out)
		})
	}
}

func Test_structID(t *testing.T) {
	test := &MyThing{field: "xxx"}
	s := structID(test)
	assert.Equal(t, "pkg.package-operator.run/cardboard/run.MyThing{field:xxx}", s)
}

func Test_methID(t *testing.T) {
	test := &MyThing{field: "xxx"}
	s := methID(test, test.private)
	assert.Equal(t, "pkg.package-operator.run/cardboard/run.MyThing{field:xxx}.private()", s)
	s = methID(test, test.privateReceiverNotPointer)
	assert.Equal(t, "pkg.package-operator.run/cardboard/run.MyThing{field:xxx}.privateReceiverNotPointer()", s)
}
