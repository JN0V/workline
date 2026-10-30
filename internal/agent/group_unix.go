//go:build !windows

package agent

import (
	"os/exec"
	"syscall"
)

// inGroup starts the agent in a process group of its own, so that a time
// cap kills it whole, the children it started too (ADR-0056): one left
// running would edit its worktree while the engine bounds it.
func inGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killGroup(cmd) }
}

// killGroup kills what is left of the agent's process group: after its
// time, or after it answered, a child of its own still running.
func killGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
