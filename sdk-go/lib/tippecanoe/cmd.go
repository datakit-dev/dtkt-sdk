package tippecanoe

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Option is a functional option that modifies a Command.
type Option func(*Command) error

type (
	// Command represents a tippecanoe invocation.
	Command struct {
		flags      []string
		files      []string
		logOutput  bool
		cleanup    []func()
		stdin      io.Reader
		outputFile string
	}
)

// NewCommand creates a new tippecanoe command by applying functional options.
// By default, it uses context.Background() and does not log output.
func NewCommand(opts ...Option) (*Command, error) {
	cmd := &Command{
		flags:     make([]string, 0),
		files:     make([]string, 0),
		logOutput: false,
	}

	for _, opt := range opts {
		if err := opt(cmd); err != nil {
			return nil, err
		}
	}
	return cmd, nil
}

func NewCommandFromConfig(cfg Config, opts ...Option) (*Command, error) {
	cmd := &Command{
		logOutput: cfg.LogOutput,
		flags:     append([]string{}, cfg.Flags...), // defensive copy
		files:     make([]string, 0),
	}
	for _, opt := range opts {
		if err := opt(cmd); err != nil {
			return nil, err
		}
	}
	return cmd, nil
}

// Run builds and executes the tippecanoe command using the stored options.
func (c *Command) Run(ctx context.Context) error {
	defer func() {
		for _, fn := range c.cleanup {
			fn()
		}
	}()

	args := append(c.flags, c.files...)
	cmd := exec.CommandContext(ctx, "tippecanoe", args...)

	if c.outputFile != "" {
		cmd.Dir = filepath.Dir(c.outputFile)
	}

	if c.stdin != nil {
		cmd.Stdin = c.stdin
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout := io.Writer(&stdoutBuf)
	stderr := io.Writer(&stderrBuf)

	if c.logOutput {
		stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tippecanoe failed: %w\nstdout:\n%s\nstderr:\n%s",
			err, stdoutBuf.String(), stderrBuf.String())
	}

	return nil
}
