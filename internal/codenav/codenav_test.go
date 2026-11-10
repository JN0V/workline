package codenav

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/packs"
)

// A C++ function defined out of its class, `Class::name`, a template's
// or a destructor, is found by its name, whole (#1064).
func TestNamedQualifiedCpp(t *testing.T) {
	src := `#include "MQTT.h"

inline void MQTTComponent::resetReconnect() {
    state = Idle;
}

template <typename T>
static std::vector<int> Queue<T>::drain(int n) {
    return {};
}

Foo::~Foo() {
}

const MQTTStatistics& ns::MQTTComponent::stats() const {
    return s;
}

void plain(int a) {
    if (a) {
        resetReconnect();
    }
}

void Queue<std::vector<int>>::push(int n) {
}
`
	lines := strings.Split(strings.TrimRight(src, "\n"), "\n")
	for name, want := range map[string][2]int{"resetReconnect": {3, 5}, "drain": {8, 10}, "~Foo": {12, 13}, "stats": {15, 17}, "plain": {19, 23}, "push": {25, 26}} {
		got := Named("MQTT_impl.h", lines, name)
		if len(got) != 1 || got[0].From != want[0] || got[0].To != want[1] {
			t.Errorf("%s: %+v, want lines %d to %d", name, got, want[0], want[1])
		}
	}
	if f, ok := Enclosing("MQTT_impl.h", lines, 4); !ok || f.Name != "resetReconnect" {
		t.Errorf("line 4 lies in %+v, want resetReconnect", f)
	}
	// A call is no definition.
	if got := Named("MQTT_impl.h", lines, "resetReconnect"); len(got) != 1 {
		t.Errorf("the call taken for a definition: %+v", got)
	}
}

// The other languages are found as before.
func TestNamedOthersUnchanged(t *testing.T) {
	for file, src := range map[string]string{
		"a.js": "function add(a, b) {\n  return a + b;\n}\n",
		"a.sh": "add() {\n  echo $(($1 + $2))\n}\n",
		"a.go": "package a\n\nfunc add(a, b int) int {\n\treturn a + b\n}\n",
	} {
		if got := Named(file, strings.Split(strings.TrimRight(src, "\n"), "\n"), "add"); len(got) != 1 {
			t.Errorf("%s: %+v", file, got)
		}
	}
}

// A project's own pack adds a language: its rows come first.
func TestUseProjectPack(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".workline", "packs"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		".workline/config.yaml":       "packs: [elixir]\n",
		".workline/packs/elixir.yaml": "code-navigation: [{name: Elixir, extensions: [.ex], definition: '^\\s*defp?\\s+(NAME)\\b', body: end}]\n",
	}
	for f, body := range files {
		if err := os.WriteFile(filepath.Join(repo, f), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	defer Use(packs.CodeRows(nil))
	if Of("lib/shop.ex") != nil {
		t.Fatal(".ex known before the pack")
	}
	if err := UseProject(repo); err != nil {
		t.Fatal(err)
	}
	lines := []string{"defmodule Shop do", "  def total(items) do", "    items", "  end", "end"}
	if f, ok := Enclosing("lib/shop.ex", lines, 3); !ok || f.Name != "total" || f.From != 2 || f.To != 4 {
		t.Errorf("Enclosing = %+v, %v", f, ok)
	}
	if Of("a.go") == nil || Of("a.sh") == nil {
		t.Error("the built-in rows after the project's")
	}
}
