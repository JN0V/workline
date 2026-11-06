#!/bin/sh
# The shape of the DomoticsCore trial's copy, where round 5's tester runs
# were recorded: components each a PlatformIO project, their native test
# suites in test/<suite>/, run by tools/workline-tests.sh into one JUnit
# report. The WebUI's platformio.ini lists the suites it runs in its
# test_filter, as the real one does; the RemoteConsole's runs them all.
#
# The runner stands in for `pio test`, which a test machine lacks: each
# project (a platformio.ini holding [env:native]) runs the suites its
# test_filter lists (each when none), and the files right under its test/;
# each `void test_x(void)` a test case. A test tools/green-tests.txt names
# passes, any other fails: the base's tests pass, a new one is red, as a
# test of a bug not fixed yet is.
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"

mkdir -p DomoticsCore-WebUI/include/DomoticsCore DomoticsCore-WebUI/test/test_webui_component_toggle \
	DomoticsCore-RemoteConsole/include/DomoticsCore DomoticsCore-RemoteConsole/test/test_remoteconsole_component \
	DomoticsCore-Core/include/DomoticsCore tools

cat > DomoticsCore-WebUI/platformio.ini <<'INI'
[env:native]
platform = native
test_framework = unity
test_build_src = true
; Only run tests designed for native (exclude hardware tests)
test_filter = test_webui_component, test_webui_registry, test_webui_leak_detection, test_webui_caching_provider, test_webui_schema_integration, test_system_header, test_schema_chunking, test_update_builder, test_webui_server_gate, test_webui_component_toggle, test_webui_auth_delay, test_webui_declared_api

[env:esp8266dev]
platform = espressif8266
test_framework = unity
test_filter = test_heap_esp8266
INI
cat > DomoticsCore-WebUI/include/DomoticsCore/WebUI.h <<'H'
#pragma once
// The WebUI component: /api/components/enable queues a switch, loop() applies it.
namespace DomoticsCore { namespace Components {
class WebUIComponent {
public:
    void begin();
    void loop();
    void shutdown();
};
} }
H
cat > DomoticsCore-WebUI/test/test_webui_component_toggle/test_webui_component_toggle.cpp <<'T'
#include <unity.h>
#include <DomoticsCore/WebUI.h>

void test_a_disable_from_the_page_waits_for_loop(void) {
    TEST_ASSERT_TRUE(true);
}

void test_an_enable_from_the_page_waits_for_loop(void) {
    TEST_ASSERT_TRUE(true);
}

int main(int argc, char** argv) {
    UNITY_BEGIN();
    RUN_TEST(test_a_disable_from_the_page_waits_for_loop);
    RUN_TEST(test_an_enable_from_the_page_waits_for_loop);
    return UNITY_END();
}
T

cat > DomoticsCore-RemoteConsole/platformio.ini <<'INI'
[env:native]
platform = native
test_framework = unity
test_build_src = true
test_ignore = test_console_capture
INI
cat > DomoticsCore-RemoteConsole/include/DomoticsCore/RemoteConsole.h <<'H'
#pragma once
// The remote console: begin() returns early when disabled; shutdown()
// removes loggerCallbackId_.
namespace DomoticsCore { namespace Components {
struct RemoteConsoleConfig { bool enabled = true; };
class RemoteConsoleComponent {
public:
    explicit RemoteConsoleComponent(const RemoteConsoleConfig& c);
    void begin();
    void shutdown();
private:
    int loggerCallbackId_ = 0;
};
} }
H
cat > DomoticsCore-RemoteConsole/test/test_remoteconsole_component/test_remoteconsole_component.cpp <<'T'
#include <unity.h>
#include <DomoticsCore/RemoteConsole.h>

void test_console_starts(void) {
    TEST_ASSERT_TRUE(true);
}

int main(int argc, char** argv) {
    UNITY_BEGIN();
    RUN_TEST(test_console_starts);
    return UNITY_END();
}
T
cat > DomoticsCore-Core/include/DomoticsCore/Logger.h <<'H'
#pragma once
// LoggerCallbacks::addCallback() hands out ids from a counter starting at 0.
H

cat > tools/green-tests.txt <<'G'
DomoticsCore-WebUI/test_webui_component_toggle test_a_disable_from_the_page_waits_for_loop
DomoticsCore-WebUI/test_webui_component_toggle test_an_enable_from_the_page_waits_for_loop
DomoticsCore-RemoteConsole/test_remoteconsole_component test_console_starts
G
cat > tools/workline-tests.sh <<'RUN'
#!/bin/sh
# tools/workline-tests.sh <report>: every native PlatformIO suite, in one
# JUnit report, its cases named <component>/<suite>; exit 1 if one fails.
report=$1
root=$(git rev-parse --show-toplevel)
cd "$root" || exit 1
case $report in /*) ;; *) report="$root/$report" ;; esac
mkdir -p "$(dirname "$report")"
failed=0
cases() { # cases <classname> <file>...: a testcase for each test function
	class=$1
	shift
	for f in "$@"; do
		[ -f "$f" ] || continue
		for t in $(sed -n 's/^\(static \)\{0,1\}void \(test_[A-Za-z0-9_]*\) *(.*/\2/p' "$f"); do
			if grep -qx "$class $t" tools/green-tests.txt; then
				echo "<testcase classname=\"$class\" name=\"$t\" time=\"0.01\"/>"
			else
				echo "<testcase classname=\"$class\" name=\"$t\" time=\"0.01\"><failure message=\"$f failed\"/></testcase>"
				echo 1 > "$report.failed"
			fi
		done
	done
}
rm -f "$report.failed"
{
	echo '<?xml version="1.0" encoding="UTF-8"?>'
	echo '<testsuites>'
	for ini in $(git ls-files '*platformio.ini'); do
		grep -q '^\[env:native\]' "$ini" || continue
		proj=$(dirname "$ini")
		filter=$(sed -n '/^\[env:native\]/,/^\[/s/^test_filter *= *//p' "$ini" | tr -d ' ')
		ignore=$(sed -n '/^\[env:native\]/,/^\[/s/^test_ignore *= *//p' "$ini" | tr -d ' ')
		echo "<testsuite name=\"$proj\">"
		cases "$proj/test" "$proj"/test/*.cpp
		for d in "$proj"/test/*/; do
			[ -d "$d" ] || continue
			suite=$(basename "$d")
			if [ -n "$filter" ]; then
				case ",$filter," in *",$suite,"*) ;; *) continue ;; esac
			fi
			case ",$ignore," in *",$suite,"*) continue ;; esac
			cases "$proj/$suite" "$d"*.cpp
		done
		echo '</testsuite>'
	done
	echo '</testsuites>'
} > "$report"
[ -f "$report.failed" ] && failed=1
rm -f "$report.failed"
exit $failed
RUN
chmod +x tools/workline-tests.sh
printf '# DomoticsCore\n\nComponents for ESP32 and ESP8266, built and tested natively with PlatformIO.\n' > README.md
git add . && git commit -q -m "feat: the components and their native tests"
