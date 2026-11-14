//go:build windows

package forge

import "os/exec"

// inGroup leaves cmd as it starts on Windows; WaitDelay bounds the wait.
func inGroup(*exec.Cmd) {}
