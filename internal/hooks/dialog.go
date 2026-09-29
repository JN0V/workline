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

// notifySend is run by its absolute path, as zenity is.
const notifySend = "/usr/bin/notify-send"

// notify shows a desktop notification that stays until dismissed, so a
// question asked in a corner of the editor is not missed. It only informs:
// where it cannot be shown, nothing changes.
func notify(summary, body string) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return
	}
	exec.Command(notifySend, "--app-name", "workline", "--urgency", "critical", "--icon", "dialog-question", "workline: "+summary, body).Run()
}
