package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
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
	Timeout  time.Duration
	Interval time.Duration
}

// Sets defaults on the waiter config.
func (c *WaiterConfig) Default() {
	if c.Timeout == 0 {
		c.Timeout = WaiterDefaultTimeout
	}
	if c.Interval == 0 {
		c.Interval = WaiterDefaultInterval
	}
}

type WaitOption interface {
	ApplyToWaiterConfig(c *WaiterConfig)
}

// Waiter implements functions to block till kube objects are in a certain state.
type Waiter struct {
	client client.Client
	scheme *runtime.Scheme
	config WaiterConfig
}

// Creates a new Waiter instance.
func NewWaiter(
	client client.Client, scheme *runtime.Scheme,
	defaultOpts ...WaitOption,
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
	GK schema.GroupKind
}

func (e *UnknownTypeError) Error() string {
	return fmt.Sprintf("unknown type: %s", e.GK)
}

// Waits for an object to be considered available.
func (w *Waiter) WaitForReadiness(
	ctx context.Context, object client.Object, opts ...WaitOption,
) error {
	gvk, err := apiutil.GVKForObject(object, w.scheme)
	if err != nil {
		return fmt.Errorf("could not determine GVK for object: %w", err)
	}

	// TODO:
	// Extend by other common types and open up to
	// register new types and readiness functions.
	gk := gvk.GroupKind()
	switch gk {
	case schema.GroupKind{
		Kind: "Deployment", Group: "apps"}:
		return w.WaitForCondition(ctx, object, "Available", metav1.ConditionTrue, opts...)

	case schema.GroupKind{
		Kind: "CustomResourceDefinition", Group: "apiextensions.k8s.io"}:
		return w.WaitForCondition(ctx, object, "Established", metav1.ConditionTrue, opts...)

	default:
		return &UnknownTypeError{GK: gk}
	}
}

// Waits for an object to report the given condition with given status.
// Takes observedGeneration into account when present on the object.
// observedGeneration may be reported on the condition or under .status.observedGeneration.
func (w *Waiter) WaitForCondition(
	ctx context.Context, object client.Object,
	conditionType string, conditionStatus metav1.ConditionStatus,
	opts ...WaitOption,
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
	opts ...WaitOption,
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
	log.Info(fmt.Sprintf("waiting %s on %s %s %s...",
		c.Timeout, gvk, key, waitReason))

	return wait.PollUntilContextTimeout(ctx, c.Interval, c.Timeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = w.client.Get(ctx, client.ObjectKeyFromObject(object), object)
			if err != nil {
				//nolint:nilerr // retry on transient errors
				return false, nil
			}

			return checkFn(object)
		},
	)
}

// Wait for an object to not exist anymore.
func (w *Waiter) WaitToBeGone(
	ctx context.Context, object client.Object,
	checkFn func(obj client.Object) (done bool, err error),
	opts ...WaitOption,
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
	log.Info(fmt.Sprintf("waiting %s for %s %s to be gone...",
		c.Timeout, gvk, key))

	return wait.PollUntilContextTimeout(
		ctx, c.Interval, c.Timeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = w.client.Get(ctx, client.ObjectKeyFromObject(object), object)
			if errors.IsNotFound(err) {
				return true, nil
			}
			if err != nil {
				//nolint:nilerr // retry on transient errors
				return false, nil
			}

			return checkFn(object)
		},
	)
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
