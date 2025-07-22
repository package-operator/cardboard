package wait

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	WaiterDefaultTimeout  = 60 * time.Second
	WaiterDefaultInterval = time.Second
)

type WaiterConfig struct {
	Timeout    time.Duration
	Interval   time.Duration
	KnownTypes map[schema.GroupVersionKind]TypeWaitFn
}

// Sets defaults on the waiter config.
func (c *WaiterConfig) Default() {
	if c.Timeout == 0 {
		c.Timeout = WaiterDefaultTimeout
	}
	if c.Interval == 0 {
		c.Interval = WaiterDefaultInterval
	}
	if c.KnownTypes == nil {
		c.KnownTypes = map[schema.GroupVersionKind]TypeWaitFn{}
	}
	c.KnownTypes[schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "Deployment",
		Version: "v1",
	}] = func(ctx context.Context, w *Waiter, object client.Object, opts ...Option) error {
		return w.WaitForCondition(ctx, object, "Available", metav1.ConditionTrue, opts...)
	}
	c.KnownTypes[schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "apiextensions.k8s.io",
		Version: "v1",
	}] = func(ctx context.Context, w *Waiter, object client.Object, opts ...Option) error {
		return w.WaitForCondition(ctx, object, "Established", metav1.ConditionTrue, opts...)
	}
}

type Option interface {
	ApplyToWaiterConfig(c *WaiterConfig)
}

// Waiter implements functions to block till kube objects are in a certain state.
type Waiter struct {
	client client.Reader
	scheme *runtime.Scheme
	config WaiterConfig
}

type TypeWaitFn func(ctx context.Context, w *Waiter, object client.Object, opts ...Option) error

// Creates a new Waiter instance.
func NewWaiter(
	client client.Reader, scheme *runtime.Scheme,
	defaultOpts ...Option,
) *Waiter {
	w := &Waiter{
		client: client,
		scheme: scheme,
	}

	for _, opt := range defaultOpts {
		opt.ApplyToWaiterConfig(&w.config)
	}
	w.config.Default()
	return w
}

// UnknownTypeError is returned when the given GroupKind is not registered.
type UnknownTypeError struct {
	GVK schema.GroupVersionKind
}

func (e *UnknownTypeError) Error() string {
	return fmt.Sprintf("unknown type: %s", e.GVK)
}

// Waits for an object to be considered available.
func (w *Waiter) WaitForReadiness(
	ctx context.Context, object client.Object, opts ...Option,
) error {
	gvk, err := apiutil.GVKForObject(object, w.scheme)
	if err != nil {
		return fmt.Errorf("could not determine GVK for object: %w", err)
	}

	fn, ok := w.config.KnownTypes[gvk]
	if !ok {
		return &UnknownTypeError{GVK: gvk}
	}

	return fn(ctx, w, object, opts...)
}

// Waits for an object to report the given condition with given status.
// Takes observedGeneration into account when present on the object.
// observedGeneration may be reported on the condition or under .status.observedGeneration.
func (w *Waiter) WaitForCondition(
	ctx context.Context, object client.Object,
	conditionType string, conditionStatus metav1.ConditionStatus,
	opts ...Option,
) error {
	return w.WaitForObject(
		ctx, object,
		fmt.Sprintf("to report condition %q=%q", conditionType, conditionStatus),
		func(obj client.Object) (done bool, err error) {
			return checkObjectCondition(obj, conditionType, conditionStatus, w.scheme)
		}, opts...)
}

// Wait for an object to match a check function.
func (w *Waiter) WaitForObject(
	ctx context.Context, object client.Object, waitReason string,
	checkFn func(obj client.Object) (done bool, err error),
	opts ...Option,
) error {
	log := logr.FromContextOrDiscard(ctx)

	c := w.config
	for _, opt := range opts {
		opt.ApplyToWaiterConfig(&c)
	}

	gvk, err := apiutil.GVKForObject(object, w.scheme)
	if err != nil {
		return err
	}

	key := client.ObjectKeyFromObject(object)
	text := fmt.Sprintf("waiting %s on %s %s %s",
		c.Timeout, gvk, key, waitReason)
	log.Info(text + "...")

	err = wait.PollUntilContextTimeout(ctx, c.Interval, c.Timeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = w.client.Get(ctx, client.ObjectKeyFromObject(object), object)
			if err != nil {
				log.Info("waiting for object errored", "key", key, "err", err)
				return false, nil
			}

			return checkFn(object)
		},
	)
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("timeout %s: %w ", text, err)
	}
	return err
}

// Wait for an object to not exist anymore.
func (w *Waiter) WaitToBeGone(
	ctx context.Context, object client.Object,
	checkFn func(obj client.Object) (done bool, err error),
	opts ...Option,
) error {
	log := logr.FromContextOrDiscard(ctx)

	c := w.config
	for _, opt := range opts {
		opt.ApplyToWaiterConfig(&c)
	}

	gvk, err := apiutil.GVKForObject(object, w.scheme)
	if err != nil {
		return err
	}

	key := client.ObjectKeyFromObject(object)
	text := fmt.Sprintf("waiting %s for %s %s to be gone",
		c.Timeout, gvk, key)
	log.Info(text + "...")

	err = wait.PollUntilContextTimeout(
		ctx, c.Interval, c.Timeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = w.client.Get(ctx, client.ObjectKeyFromObject(object), object)
			if apimachineryerrors.IsNotFound(err) {
				return true, nil
			}
			if err != nil {
				//nolint:nilerr // retry on transient errors
				return false, nil
			}

			return checkFn(object)
		},
	)
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("timeout %s: %w", text, err)
	}
	return err
}

// Check if a object condition is in a certain state.
// Will respect .status.observedGeneration and .status.conditions[].observedGeneration.
func checkObjectCondition(
	obj client.Object, conditionType string,
	conditionStatus metav1.ConditionStatus,
	scheme *runtime.Scheme,
) (done bool, err error) {
	unstrObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		unstrObj = &unstructured.Unstructured{}
		if err := scheme.Convert(obj, unstrObj, nil); err != nil {
			return false, fmt.Errorf("can't convert to unstructured: %w", err)
		}
	}

	if observedGen, ok, err := unstructured.NestedInt64(
		unstrObj.Object, "status", "observedGeneration"); err != nil {
		return false, fmt.Errorf("could not access .status.observedGeneration: %w", err)
	} else if ok && observedGen != obj.GetGeneration() {
		// Object status outdated
		return false, nil
	}

	conditionsRaw, ok, err := unstructured.NestedFieldNoCopy(
		unstrObj.Object, "status", "conditions")
	if err != nil {
		return false, fmt.Errorf("could not access .status.conditions: %w", err)
	}
	if !ok {
		// no conditions reported
		return false, nil
	}

	// Press into metav1.Condition scheme to be able to work typed.
	conditionsJSON, err := json.Marshal(conditionsRaw)
	if err != nil {
		return false, fmt.Errorf("could not marshal conditions into JSON: %w", err)
	}
	var conditions []metav1.Condition
	if err := json.Unmarshal(conditionsJSON, &conditions); err != nil {
		return false, fmt.Errorf("could not unmarshal conditions: %w", err)
	}

	// Check conditions
	condition := meta.FindStatusCondition(conditions, conditionType)
	if condition == nil {
		// no such condition
		return false, nil
	}

	if condition.ObservedGeneration != 0 &&
		condition.ObservedGeneration != obj.GetGeneration() {
		// Condition outdated
		return false, nil
	}

	return condition.Status == conditionStatus, nil
}
