//go:build !windows

package forge

import (
	"os/exec"
	"syscall"
)

// inGroup starts cmd in a process group of its own, killed whole when its
// context ends, so a child holding its output does not outlive the timeout.
// (internal/agent has the same helper; it imports this package.)
func inGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
