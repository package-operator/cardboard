package run

import "fmt"

// used to wrap unexpected panics.
type internalPanicedError struct {
	ID    string // (optional) call id, e.g. object:func
	Obj   any    // object thrown by panic
	Stack string
}

func (p *internalPanicedError) Error() string {
	if len(p.ID) > 0 {
		return fmt.Sprintf("Panic in call %q: %v\n%v", p.ID, p.Obj, p.Stack)
	}
	return fmt.Sprintf("Panic: %v\n%v", p.Obj, p.Stack)
}

// UnknownTargetError is raised with no target under the given name could be found.
type UnknownTargetError struct {
	ID string
}

func (t *UnknownTargetError) Error() string {
	return fmt.Sprintf("unknown target: %q", t.ID)
}
