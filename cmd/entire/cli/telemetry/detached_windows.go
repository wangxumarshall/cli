//go:build windows

package telemetry

import (
	"context"
	"io"
	"os"
	"os/exec"
	"syscall"
)

// spawnDetachedAnalytics spawns a detached subprocess to send analytics.
// On Windows, this uses CREATE_NEW_PROCESS_GROUP and DETACHED_PROCESS flags
// so the subprocess continues after the parent exits.
func spawnDetachedAnalytics(payloadJSON string) {
	executable, err := os.Executable()
	if err != nil {
		return
	}

	cmd := exec.CommandContext(context.Background(), executable, "__send_analytics", payloadJSON)

	// Detach from parent process so subprocess survives parent exit
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	// Use temp dir since "/" doesn't exist on Windows
	cmd.Dir = os.TempDir()

	// Inherit environment (may be needed for network config)
	cmd.Env = os.Environ()

	// Discard stdout/stderr to prevent output leaking to parent's terminal
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	// Start the process (non-blocking)
	if err := cmd.Start(); err != nil {
		return
	}

	// Release the process so it can run independently
	//nolint:errcheck // Best effort - process should continue regardless
	_ = cmd.Process.Release()
}
