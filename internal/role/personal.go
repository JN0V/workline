package role

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"go.yaml.in/yaml/v3"
)

// PersonalConfig is the file a person keeps what they want wherever they
// work: <config folder>/workline/config.yaml (~/.config on Linux), never in
// a repository.
func PersonalConfig() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "workline", "config.yaml")
}

// PersonalSettings are a person's own settings for a role, as their config
// holds them under `roles: {<role>: {settings: …}}`; nil when it says
// nothing of the role, or there is no such file.
func PersonalSettings(name string) (map[string]any, error) {
	path := PersonalConfig()
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var c struct {
		Roles map[string]struct {
			Settings map[string]any `yaml:"settings"`
		} `yaml:"roles"`
	}
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	return c.Roles[name].Settings, nil
}

// WithPersonal lays a person's own settings over the role's merged ones,
// only those the role lets a person set (`personal` in role.yaml): a list
// adds to the project's, any other value takes its place. The keys a person
// may not set are returned, sorted, and left out.
func (r *Role) WithPersonal(merged, personal map[string]any) (map[string]any, []string) {
	if len(personal) == 0 {
		return merged, nil
	}
	out := MergeSettings(merged, nil)
	keys := make([]string, 0, len(personal))
	for k := range personal {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var refused []string
	for _, k := range keys {
		if !slices.Contains(r.Personal, k) {
			refused = append(refused, k)
			continue
		}
		add, isList := personal[k].([]any)
		have, _ := out[k].([]any)
		if !isList {
			out[k] = cloneSetting(personal[k])
			continue
		}
		list := slices.Clone(have)
		for _, v := range add {
			if !slices.ContainsFunc(list, func(x any) bool { return fmt.Sprint(x) == fmt.Sprint(v) }) {
				list = append(list, cloneSetting(v))
			}
		}
		out[k] = list
	}
	return out, refused
}
