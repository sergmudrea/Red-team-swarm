package agent

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"time"
)

// ExecuteCommand runs a shell command with the given timeout.
// It automatically selects the appropriate shell (sh on Unix, cmd on Windows).
func ExecuteCommand(cmd string, timeout time.Duration) (stdout, stderr string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var shell string
	var shellArgs []string
	if runtime.GOOS == "windows" {
		shell = "cmd"
		shellArgs = []string{"/c"}
	} else {
		shell = "sh"
		shellArgs = []string{"-c"}
	}

	c := exec.CommandContext(ctx, shell, append(shellArgs, cmd)...)

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
