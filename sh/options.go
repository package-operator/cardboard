package sh

import (
	"io"
	"log/slog"
)

type RunnerOption interface {
	ApplyToRunner(r *Runner)
}

type WithLogger struct{ *slog.Logger }

func (l WithLogger) ApplyToRunner(r *Runner) {
	r.logger = l.Logger
}

type WithEnvironment map[string]string

func (e WithEnvironment) ApplyToRunner(r *Runner) {
	r.env = e
}

type WithWorkDir string

func (wd WithWorkDir) ApplyToRunner(r *Runner) {
	r.workDir = string(wd)
}

type WithStdout struct{ io.Writer }

func (stdout WithStdout) ApplyToRunner(r *Runner) {
	r.stdout = stdout.Writer
}

type WithStderr struct{ io.Writer }

func (stderr WithStderr) ApplyToRunner(r *Runner) {
	r.stderr = stderr.Writer
}
