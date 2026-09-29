package hooks

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// zenity is run by its absolute path, never looked up on PATH, where an
// agent could put a program answering for the person (ADR-0008).
const zenity = "/usr/bin/zenity"

// desktopDialog asks in a window on the desktop, when there is one to show
// it on. macOS and kdialog are not tried yet.
func desktopDialog() (func(question, title string) (string, error), bool) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return nil, false
	}
	if _, err := os.Stat(zenity); err != nil {
		return nil, false
	}
	return func(question, title string) (string, error) {
		out, err := exec.Command(zenity, "--question", "--no-markup", "--title", "workline",
			"--text", title, "--ok-label", "Push", "--cancel-label", "Stop", "--extra-button", "View",
			"--timeout", fmt.Sprint(int(answerTimeout.Seconds()))).Output()
		var exit *exec.ExitError
		switch {
		case err == nil:
			return "y", nil
		case errors.As(err, &exit) && exit.ExitCode() == 1 && strings.TrimSpace(string(out)) == "View":
			return "v", nil
		case errors.As(err, &exit) && exit.ExitCode() == 1:
			return "", nil
		case errors.As(err, &exit) && exit.ExitCode() == 5:
			return "", errors.New("no answer in time")
		default:
			return "", err
		}
	}, true
}
