package run

import (
	"fmt"
	"runtime"
)

func decorateWithCallingSourceLine(err error) error {
	if err == nil {
		return nil
	}

	// make sure to change this number if calls get introduced to the stack or make it parametrizable by the caller
	// TODO: unit test that serves as a regression test for the correct "depth"
	_, fi, line, ok := runtime.Caller(2)
	if !ok {
		return fmt.Errorf("[source code not found] %w", err)
	}
	return fmt.Errorf("[%s#L%d] %w", fi, line, err)
}
