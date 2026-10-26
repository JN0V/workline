package linters

import (
	"path"
	"regexp"
	"strings"
)

// commands are the commands a shell text runs, each its words joined by
// a space, quotes taken off: split on unquoted newlines, ;, & and |, its
// comments left out, and the commands that only name a tool (installs,
// downloads, files moved, words printed) left out too.
func commands(text string) []string {
	var out, words []string
	var word strings.Builder
	inWord := false
	end := func() {
		if inWord {
			words = append(words, word.String())
			word.Reset()
			inWord = false
		}
	}
	flush := func() {
		end()
		if len(words) > 0 {
			words[0] = strings.TrimLeft(words[0], "@-+") // make's prefixes
		}
		if runs(words) {
			out = append(out, strings.Join(words, " "))
		}
		words = nil
	}
	rs := []rune(text)
	for i := 0; i < len(rs); i++ {
		switch c := rs[i]; {
		case c == '\\' && i+1 < len(rs):
			i++
			if rs[i] != '\n' { // a line continued
				word.WriteRune(rs[i])
				inWord = true
			}
		case c == '\'':
			inWord = true
			for i++; i < len(rs) && rs[i] != '\''; i++ {
				word.WriteRune(rs[i])
			}
		case c == '"':
			inWord = true
			for i++; i < len(rs) && rs[i] != '"'; i++ {
				if rs[i] == '\\' && i+1 < len(rs) {
					i++
				}
				word.WriteRune(rs[i])
			}
		case c == '#' && !inWord:
			for i < len(rs) && rs[i] != '\n' {
				i++
			}
			flush()
		case c == '\n' || c == ';' || c == '&' || c == '|':
			flush()
		case c == ' ' || c == '\t' || c == '\r':
			end()
		default:
			word.WriteRune(c)
			inWord = true
		}
	}
	flush()
	return out
}

var (
	// names: a command that names a tool and runs none.
	names = set("curl", "wget", "install", "chmod", "mv", "cp", "ln", "tar", "unzip", "rm", "mkdir", "echo", "printf", "cat", "cd", "export", "touch")
	// managers install a tool with a verb among their first two words
	// (go install, rustup component add, pio pkg install), flags skipped.
	managers = set("pip", "pip3", "pipx", "uv", "npm", "pnpm", "yarn", "bun", "go", "cargo", "rustup", "brew", "apt-get", "apt",
		"gem", "apk", "dnf", "yum", "choco", "scoop", "winget", "snap", "mise", "asdf", "aqua", "conda", "poetry", "docker", "pio", "platformio")
	installs = set("install", "add", "get", "use", "i", "pull")
	assign   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	python   = regexp.MustCompile(`^(python[0-9.]*|py)$`)
)

func set(words ...string) map[string]bool {
	m := map[string]bool{}
	for _, w := range words {
		m[w] = true
	}
	return m
}

// runs says whether a command's words run what they name: not an
// install (`python -m pip install`, `apt-get -y install`), a download or
// an echo.
func runs(words []string) bool {
	prefix := false // sudo -E, env -i: their flags are theirs
	for len(words) > 0 {
		w := strings.TrimLeft(words[0], "@-+") // make's @, - and +
		if w == "sudo" || w == "env" || w == "time" || assign.MatchString(w) || (prefix && strings.HasPrefix(words[0], "-")) {
			prefix = prefix || w == "sudo" || w == "env" || w == "time"
			words = words[1:]
			continue
		}
		break
	}
	if len(words) == 0 {
		return false
	}
	first := path.Base(strings.TrimLeft(words[0], "@-+"))
	if python.MatchString(first) && len(words) > 2 && words[1] == "-m" {
		first, words = words[2], words[2:]
	}
	if names[first] {
		return false
	}
	if managers[first] {
		seen := 0
		for _, w := range words[1:] {
			if strings.HasPrefix(w, "-") {
				continue
			}
			if installs[w] {
				return false
			}
			if seen++; seen == 2 {
				break
			}
		}
	}
	return true
}
