package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"
)

// Seen is where a machine keeps the last exact model that answered for each
// model asked: an alias floats to a new generation on its own, and the
// evaluation's scores are worth something only for the model that earned
// them (docs/adr/0004-follow-model-aliases-and-measure.md).
// WORKLINE_MODELS_SEEN names another file.
func Seen() string {
	if f := os.Getenv("WORKLINE_MODELS_SEEN"); f != "" {
		return f
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "workline", "models-seen.yaml")
}

type seenModel struct {
	Model string `yaml:"model"`
	Since string `yaml:"since"`
}

// Notice records the model that answered call in file, and says so when it
// is not the one that answered the same question last time on this machine.
// A call that did not say what it asked or what answered is not recorded. It
// never fails a run: a file it cannot read or write only means no notice.
func Notice(file string, call Call) string {
	if file == "" || call.Asked == "" || call.Model == "" {
		return ""
	}
	seen := map[string]seenModel{}
	if data, err := os.ReadFile(file); err == nil {
		_ = yaml.Unmarshal(data, &seen)
	}
	key := call.Agent + ":" + call.Asked
	was, known := seen[key]
	if known && was.Model == call.Model {
		return ""
	}
	seen[key] = seenModel{Model: call.Model, Since: time.Now().UTC().Format("2006-01-02")}
	if data, err := yaml.Marshal(seen); err == nil && os.MkdirAll(filepath.Dir(file), 0o755) == nil {
		tmp := file + ".tmp"
		if os.WriteFile(tmp, data, 0o644) == nil {
			_ = os.Rename(tmp, file)
		}
	}
	if !known {
		return "" // the first model seen is not a change
	}
	return fmt.Sprintf("%s %q is now answered by %s, no longer %s (seen since %s): the evaluation's scores were earned by the old one — run it (docs/spec/conformance.md)",
		call.Agent, call.Asked, call.Model, was.Model, was.Since)
}
