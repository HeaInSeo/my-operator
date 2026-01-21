package kubeutil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/yeongki/my-operator/pkg/slo"
)

// CmdRunner abstracts command execution (stdout-only on success).
type CmdRunner interface {
	Run(ctx context.Context, logger slo.Logger, cmd *exec.Cmd) (string, error)
}

// DefaultRunner executes commands and returns stdout.
// On error, includes stderr+stdout in the returned error.
type DefaultRunner struct{}

func (DefaultRunner) Run(ctx context.Context, logger slo.Logger, cmd *exec.Cmd) (string, error) {
	logger = slo.NewLogger(logger)

	// Ensure ctx cancellation works even if the caller constructed cmd without context.
	// We rebuild the command using exec.CommandContext but preserve args, dir, stdin.
	// Note: If cmd.Path is empty, cmd.Args[0] is used; but normally exec.Command sets Path.
	path := cmd.Path
	if path == "" && len(cmd.Args) > 0 {
		path = cmd.Args[0]
	}
	args := []string{}
	if len(cmd.Args) > 1 {
		args = cmd.Args[1:]
	}

	c2 := exec.CommandContext(ctx, path, args...)
	c2.Dir = cmd.Dir
	c2.Stdin = cmd.Stdin
	c2.Env = cmd.Env
	if len(c2.Env) == 0 {
		c2.Env = append(os.Environ(), "GO111MODULE=on")
	} else {
		c2.Env = append(c2.Env, "GO111MODULE=on")
	}

	command := strings.Join(c2.Args, " ")
	logger.Logf("running: %q", command)

	var stdout, stderr bytes.Buffer
	c2.Stdout = &stdout
	c2.Stderr = &stderr

	err := c2.Run()
	outStr := stdout.String()
	errStr := stderr.String()

	if err != nil {
		combined := strings.TrimSpace(errStr + "\n" + outStr)
		return outStr, fmt.Errorf("%q failed: %s: %w", command, combined, err)
	}
	return outStr, nil
}
