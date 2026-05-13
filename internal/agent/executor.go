// Package agent implements the agent-side logic (command execution, SOCKS5 proxy, self-destruct).
package agent

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// ExecuteCommand runs a shell command with the given timeout.
// It returns captured stdout, stderr, and any error that occurred during execution.
func ExecuteCommand(cmd string, timeout time.Duration) (stdout, stderr string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	c := exec.CommandContext(ctx, "sh", "-c", cmd)

	var outBuf, errBuf bytes.Buffer
	c.Stdout = &outBuf
	c.Stderr = &errBuf

	err = c.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if ctx.Err() == context.DeadlineExceeded {
		err = ctx.Err()
	}
	return
}
