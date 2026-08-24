package adapter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const maxOutput = 64 * 1024

// Spec is one command to execute.
type Spec struct {
	Command []string
	Dir     string
	Env     map[string]string
}

// Outcome is the result of one adapter invocation.
type Outcome struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Adapter runs a journey step.
type Adapter interface {
	Run(ctx context.Context, spec Spec) (Outcome, error)
}

// Shell executes argv directly. There is no shell interpolation.
type Shell struct{}

func (Shell) Run(ctx context.Context, spec Spec) (Outcome, error) {
	if len(spec.Command) == 0 {
		return Outcome{}, fmt.Errorf("empty command")
	}
	cmd := exec.CommandContext(ctx, spec.Command[0], spec.Command[1:]...)
	if spec.Dir != "" {
		cmd.Dir = spec.Dir
	}
	cmd.Env = mergeEnv(os.Environ(), spec.Env)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &limitedWriter{buf: &stdout, max: maxOutput}
	cmd.Stderr = &limitedWriter{buf: &stderr, max: maxOutput}
	err := cmd.Run()
	out := Outcome{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if err == nil {
		return out, nil
	}
	if exit, ok := err.(*exec.ExitError); ok {
		out.ExitCode = exit.ExitCode()
		return out, nil
	}
	return out, err
}

func mergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	env := make([]string, 0, len(base)+len(extra))
	seen := make(map[string]struct{}, len(extra))
	for k := range extra {
		seen[strings.ToUpper(k)] = struct{}{}
	}
	for _, kv := range base {
		key, _, _ := strings.Cut(kv, "=")
		if _, override := extra[key]; override {
			continue
		}
		if _, skip := seen[strings.ToUpper(key)]; skip {
			continue
		}
		env = append(env, kv)
	}
	for k, v := range extra {
		env = append(env, k+"="+v)
	}
	return env
}

type limitedWriter struct {
	buf *bytes.Buffer
	max int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	remain := w.max - w.buf.Len()
	if remain <= 0 {
		return len(p), nil
	}
	if len(p) > remain {
		_, err := w.buf.Write(p[:remain])
		return len(p), err
	}
	return w.buf.Write(p)
}
