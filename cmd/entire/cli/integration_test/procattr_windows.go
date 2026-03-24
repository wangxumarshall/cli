//go:build integration && windows

package integration

import (
	"os/exec"
	"syscall"
)

// detachFromTerminal configures the command to not inherit the parent console,
// preventing interactive prompts from hanging tests.
func detachFromTerminal(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
