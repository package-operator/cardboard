package run

// MustError wraps errors from MustX() functions.
type MustError struct {
	err error
}

func (e *MustError) Error() string {
	return e.err.Error()
}

// Must panics when err is not nil.
func Must(err error) {
	if err != nil {
		panic(&MustError{err: err})
	}
}

// Must1 panics when err is not nil and support an additional parameter.
func Must1(_ any, err error) {
	Must(err)
}

// Must2 panics when err is not nil and support two additional parameters.
func Must2(_, _ any, err error) {
	Must(err)
}

// Must3 panics when err is not nil and support tree additional parameters.
func Must3(_, _, _ any, err error) {
	Must(err)
}

// Must4 panics when err is not nil and support four additional parameters.
func Must4(_, _, _, _ any, err error) {
	Must(err)
}
