package wait

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

type WithInterval time.Duration

func (i WithInterval) ApplyToWaiterConfig(c *WaiterConfig) {
	c.Interval = time.Duration(i)
}

type WithTimeout time.Duration

func (t WithTimeout) ApplyToWaiterConfig(c *WaiterConfig) {
	c.Timeout = time.Duration(t)
}

type WithTypeWaitFunctions map[schema.GroupVersionKind]TypeWaitFn

func (fn WithTypeWaitFunctions) ApplyToWaiterConfig(c *WaiterConfig) {
	c.KnownTypes = fn
}
