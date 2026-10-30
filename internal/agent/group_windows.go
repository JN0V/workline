//go:build windows

package agent

import "os/exec"

// inGroup leaves the agent as it starts on Windows: its time cap kills
// the process itself, not the children it started.
func inGroup(*exec.Cmd) {}

func killGroup(*exec.Cmd) error { return nil }
