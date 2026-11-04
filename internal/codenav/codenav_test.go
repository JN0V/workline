package codenav

import (
	"strings"
	"testing"
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
`
	lines := strings.Split(strings.TrimRight(src, "\n"), "\n")
	for name, want := range map[string][2]int{"resetReconnect": {3, 5}, "drain": {8, 10}, "~Foo": {12, 13}, "stats": {15, 17}, "plain": {19, 23}} {
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
