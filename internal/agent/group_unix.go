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

// InGroup starts cmd in a process group of its own, killed whole when its
// context ends: the processes a test command started with it (#1197). A
// daemon that detaches into a session of its own, as Gradle's, is not.
func InGroup(cmd *exec.Cmd) { inGroup(cmd) }

// KillGroup kills what is left of cmd's process group once cmd ended: a
// daemon its command forked and left running (#1335). A daemon that
// detaches into a session of its own is out of reach.
func KillGroup(cmd *exec.Cmd) { _ = killGroup(cmd) }
