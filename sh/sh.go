// package sh provides a convenience interface to issue shell commands.
package sh

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

type Runner struct {
	logger         *slog.Logger
	env            map[string]string
	stdout, stderr io.Writer
	workDir        string
}

func New(opts ...RunnerOption) *Runner {
	r := &Runner{}
	r.apply(opts...)
	return r
}

func (r *Runner) New(opts ...RunnerOption) *Runner {
	nr := &Runner{
		logger:  r.logger,
		env:     r.env,
		stdout:  r.stdout,
		stderr:  r.stderr,
		workDir: r.workDir,
	}
	nr.apply(opts...)
	return nr
}

func (r *Runner) apply(opts ...RunnerOption) {
	for _, opt := range opts {
		opt.ApplyToRunner(r)
	}
}

func (r *Runner) Run(cmd string, args ...string) error {
	return r.run(outOrStdoutIfNil(r.stdout), outOrStderrIfNil(r.stderr), nil, cmd, args...)
}

func (r *Runner) Bash(script ...string) error {
	scriptBuf := bytes.NewBufferString(strings.Join(script, "\n"))
	return r.run(outOrStdoutIfNil(r.stdout), outOrStderrIfNil(r.stderr), scriptBuf, "bash")
}

func (r *Runner) Output(cmd string, args ...string) (string, error) {
	var out bytes.Buffer
	err := r.run(&out, outOrStderrIfNil(r.stderr), nil, cmd, args...)
	return strings.TrimRight(out.String(), "\n"), err
}

func (r *Runner) Copy(dst, src string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0o644)
}

func (r *Runner) run(stdout, stderr io.Writer, stdin io.Reader, cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Env = os.Environ()
	for k, v := range r.env {
		c.Env = append(c.Env, k+"="+v)
	}

	var stderrBuf bytes.Buffer
	if stderr == nil {
		stderr = &stderrBuf
	}

	c.Stdin = stdin
	c.Stdout = stdout
	c.Stderr = stderr
	c.Dir = r.workDir

	if r.logger != nil {
		r.logger.Info("exec", slog.String("cmd", cmd), slog.String("args", strings.Join(args, ", ")))
	}

	err := c.Run()
	if err == nil {
		return nil
	}
	if cmdRan(err) {
		code := exitStatus(err)
		if stderrBuf.Len() > 0 {
			return fmt.Errorf(`running "%s %s" failed with exit code %d: %s`, cmd, strings.Join(args, " "), code, strings.TrimRight(stderrBuf.String(), "\n"))
		}
		return fmt.Errorf(`running "%s %s" failed with exit code %d`, cmd, strings.Join(args, " "), code)
	}
	return fmt.Errorf(`failed to run "%s %s": %w`, cmd, strings.Join(args, " "), err)
}

// cmdRan examines the error to determine if it was generated as a result of a
// command running via os/exec.Command.  If the error is nil, or the command ran
// (even if it exited with a non-zero exit code), CmdRan reports true.  If the
// error is an unrecognized type, or it is an error from exec.Command that says
// the command failed to run (usually due to the command not existing or not
// being executable), it reports false.
func cmdRan(err error) bool {
	if err == nil {
		return true
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.Exited()
	}
	return false
}

type exitStatusAccessor interface {
	ExitStatus() int
}

// exitStatus returns the exit status of the error if it is an exec.ExitError
// or if it implements ExitStatus() int.
// 0 if it is nil or 1 if it is a different error.
func exitStatus(err error) int {
	if err == nil {
		return 0
	}
	//nolint:errorlint
	if e, ok := err.(exitStatusAccessor); ok {
		return e.ExitStatus()
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		if ex, ok := exitError.Sys().(exitStatusAccessor); ok {
			return ex.ExitStatus()
		}
	}
	return 1
}
