#!/bin/sh
# A component whose docs drifted the way real ones did (ADR-0014, step 1;
# docs/research/documentalist-fixes-reviewed.md). Written true, then the code
# moved on and a person moved `checked` without reading the sources, as
# DomoticsCore's 9e20ce2 did: at HEAD every doc is checked, none is suspect,
# and these stand false under it, as in DomoticsCore:
#
# - a fenced tree listing "Clock.h (524 lines)", which has 569 (NTP.h);
# - a `| File | Lines |` row saying the same (ntp/README.md);
# - "ClockWebUI.h is currently ~283 lines", which has 236 (EventBus.h);
# - "returns hardcoded "1.4.0" instead of "1.4.1"", a bug fixed (92feb08);
# - "33 tests" in a file no source of the doc names, which has 29 (test
#   counts never recounted): the agent can neither fix it nor vouch for it.
#
# Around them, what must stay: "< 800 lines" and "Hard limit: 800 lines",
# limits, not counts (hal-architecture.md); "every five minutes", the default,
# beside a code comment still saying once an hour (workline #29); "tried on a
# real ESP32 board, the router unplugged", backed only by tried.md, a record
# (d43b3f2). getting-started.md shares the version with README.md (the
# siblings 16c660b left at 1.9.0). docs/clock/events.md is true in every
# sentence: the clean control, which must end `checked`.
#
# Sources fit whole in a task (under 20,000 characters a doc), so the agent
# reads them all: what it vouches for is its own doing, not the engine's.
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
at() { export GIT_AUTHOR_DATE="$1T00:00:00Z" GIT_COMMITTER_DATE="$1T00:00:00Z"; }
lines() { test "$(wc -l < "$1")" -eq "$2" || { echo "drifted.sh: $1 has $(wc -l < "$1") lines, not $2" >&2; exit 1; }; }
mkdir -p Clock/include Clock/test Clock/examples/basic docs/clock

# clock_h <entries>: the component, its drift table that many entries long.
clock_h() {
	cat <<'CPP'
#pragma once
// Clock: keeps the device's time from NTP, corrected for the crystal's drift.
#include <stdint.h>
#include "ClockConfig.h"
#include "ClockEvents.h"

#define CLOCK_VERSION "1.4.1"

namespace clock_component {

// Sends one NTP request and reads the answer; false when none came.
bool ntpQuery(const char* server, int32_t* offsetMs);

// Publishes an event on the bus, with one number as its payload.
void publish(const char* topic, int32_t value);

class Clock {
public:
  explicit Clock(const ClockConfig& config) : config_(config) {}

  // Call once from setup(): the first sync comes at the first loop().
  void begin() {
    nextSyncAtMs_ = 0;
    failures_ = 0;
  }

  // Call from loop(): syncs when the interval is up. After a failure, a lost
  // network for one, the next try comes retryAfterMs later, until one works.
  void loop(uint32_t nowMs) {
    if (nowMs < nextSyncAtMs_) {
      return;
    }
    if (sync()) {
      failures_ = 0;
      nextSyncAtMs_ = nowMs + config_.syncIntervalMs;
    } else {
      failures_++;
      publish(clock_events::SYNC_FAILED, failures_);
      nextSyncAtMs_ = nowMs + config_.retryAfterMs;
    }
  }

  // The time, in milliseconds since the epoch, drift corrected.
  int64_t nowMs(uint32_t uptimeMs, int temperatureC) const {
    return baseMs_ + uptimeMs + driftMs(uptimeMs, temperatureC);
  }

  // How many syncs failed in a row.
  int failures() const { return failures_; }

private:
  bool sync() {
    int32_t offset = 0;
    if (!ntpQuery(config_.server, &offset)) {
      return false;
    }
    baseMs_ += offset;
    publish(clock_events::SYNCED, offset);
    return true;
  }

  // The drift since the last sync, from the table below, by temperature.
  static int64_t driftMs(uint32_t uptimeMs, int temperatureC);

  ClockConfig config_;
  int64_t baseMs_ = 0;
  uint32_t nextSyncAtMs_ = 0;
  int failures_ = 0;
};

// Drift of the crystal in parts per million, by temperature, from -40 C in
// steps of a quarter degree: measured on the reference board.
static const int16_t kDriftPpm[] = {
CPP
	i=0
	while [ "$i" -lt "$1" ]; do
		t=$((i / 4 - 40))
		echo "  $(( (t - 25) * (t - 25) / 40 - 20 )),"
		i=$((i + 1))
	done
	cat <<'CPP'
};

inline int64_t Clock::driftMs(uint32_t uptimeMs, int temperatureC) {
  int i = (temperatureC + 40) * 4;
  const int n = sizeof(kDriftPpm) / sizeof(kDriftPpm[0]);
  if (i < 0) {
    i = 0;
  }
  if (i >= n) {
    i = n - 1;
  }
  return (int64_t)uptimeMs * kDriftPpm[i] / 1000000;
}

}  // namespace clock_component
CPP
}

# webui_h <legacy style lines> <version expression>: the page and its version.
webui_h() {
	cat <<'CPP'
#pragma once
// The clock's page in the web UI: the time, the last offset, the failures.
#include "Clock.h"

namespace clock_component {

class ClockWebUI {
public:
  explicit ClockWebUI(const Clock& clock) : clock_(clock) {}

  // The page, served at /clock.
  const char* page() const { return kPage; }

  // The version shown in the page's footer.
CPP
	echo "  const char* getWebUIVersion() const { return $2; }"
	cat <<'CPP'

private:
  const Clock& clock_;
  static constexpr const char* kPage = R"HTML(<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>Clock</title>
CPP
	i=0
	while [ "$i" -lt "$1" ]; do
		echo ".legacy-$i { margin: ${i}px; }"
		i=$((i + 1))
	done
	cat <<'CPP'
</head>
<body>
<h1>Clock</h1>
<table>
<tr><td>Time</td><td id="time"></td></tr>
<tr><td>Last offset</td><td id="offset"></td></tr>
<tr><td>Failures in a row</td><td id="failures"></td></tr>
</table>
<label for="tz">Time zone</label>
<select id="tz">
CPP
	i=-12
	while [ "$i" -le 14 ]; do
		echo "<option value=\"$i\">UTC$( [ "$i" -ge 0 ] && echo + )$i</option>"
		i=$((i + 1))
	done
	j=0
	while [ "$j" -lt 167 ]; do
		echo "<option value=\"x$j\">zone $j</option>"
		j=$((j + 1))
	done
	cat <<'CPP'
</select>
<footer id="version"></footer>
<script src="/clock.js"></script>
</body>
</html>
)HTML";
};

}  // namespace clock_component
CPP
}

# tests_cpp <n>: the unit tests, n of them.
tests_cpp() {
	echo '#include <unity.h>'
	echo '#include "Clock.h"'
	i=1
	while [ "$i" -le "$1" ]; do
		echo "void test_clock_$i() { TEST_ASSERT_TRUE(true); }"
		i=$((i + 1))
	done
	echo 'int main() {'
	echo '  UNITY_BEGIN();'
	i=1
	while [ "$i" -le "$1" ]; do
		echo "  RUN_TEST(test_clock_$i);"
		i=$((i + 1))
	done
	echo '  return UNITY_END();'
	echo '}'
}

# The component at 1.4.1, the web UI's version hardcoded, a sync each hour.
at 2026-01-01
printf '{\n  "name": "clock",\n  "version": "1.4.1",\n  "frameworks": ["arduino"]\n}\n' > Clock/library.json
clock_h 436 > Clock/include/Clock.h
webui_h 47 '"1.4.0"' > Clock/include/ClockWebUI.h
cat > Clock/include/ClockConfig.h <<'CPP'
#pragma once
#include <stdint.h>

// What the clock is told at begin(); every field has a default.
struct ClockConfig {
  // The NTP server asked.
  const char* server = "pool.ntp.org";
  // Resync once an hour: the NTP pool asks clients not to poll more often.
  uint32_t syncIntervalMs = 3600000;
  // After a failed sync, the next try comes this much later.
  uint32_t retryAfterMs = 10000;
};
CPP
cat > Clock/include/ClockEvents.h <<'CPP'
#pragma once
// Events the clock publishes.
namespace clock_events {
// After each successful sync; payload: the offset corrected, in milliseconds.
constexpr const char* SYNCED = "clock/synced";
// When a sync fails; payload: the number of failures in a row.
constexpr const char* SYNC_FAILED = "clock/sync-failed";
}  // namespace clock_events
CPP
tests_cpp 33 > Clock/test/test_clock.cpp
cat > Clock/examples/basic/main.cpp <<'CPP'
#include <Arduino.h>
#include "Clock.h"

clock_component::Clock clock{ClockConfig{}};

void setup() { clock.begin(); }

void loop() { clock.loop(millis()); }
CPP
lines Clock/include/Clock.h 524
lines Clock/include/ClockWebUI.h 283
lines Clock/include/ClockConfig.h 12

cat > docs/clock/README.md <<'DOC'
---
sources: [Clock/library.json, Clock/include/Clock.h, Clock/include/ClockConfig.h]
checked: HEAD
---
# Clock

Keeps the device's time from NTP, corrected for the crystal's drift by
temperature.

## Version

Current version: **1.4.1** (as declared in `library.json`).

## Configuration

| Field | Default | Meaning |
|---|---|---|
| `server` | `pool.ntp.org` | the NTP server asked |
| `syncIntervalMs` | 3600000 | the clock resyncs once an hour by default |
| `retryAfterMs` | 10000 | after a failed sync, the next try comes ten seconds later |

## Source files

| File | Lines |
|---|---|
| `Clock.h` | 524 |
| `ClockConfig.h` | 12 |

Each file stays < 800 lines, the project's limit.
DOC
cat > docs/clock/project-context.md <<'DOC'
---
sources: [Clock/library.json, Clock/include/Clock.h, Clock/include/ClockWebUI.h]
checked: HEAD
---
# Clock — project context

What an agent working on the Clock component needs to know.

## Files

```
Clock/
  library.json                      # package manifest (v1.4.1)
  include/
    Clock.h           (524 lines)   # Clock class, drift table
    ClockWebUI.h                    # the web UI's page
  examples/basic/main.cpp           # begin() in setup(), loop() in loop()
```

## Size

- Target: 200-500 lines per file. Hard limit: 800 lines.
- `ClockWebUI.h` is currently ~283 lines.

## Known issues

- **Bug (minor)**: `getWebUIVersion()` returns hardcoded `"1.4.0"` instead of `"1.4.1"` -- mismatch with `library.json`.
DOC
cat > docs/clock/status.md <<'DOC'
---
sources: [Clock/include/Clock.h]
checked: HEAD
---
# Clock — where it stands

| What | Where it stands |
|---|---|
| Sync from NTP, each interval | built, tried on a real ESP32 board |
| Retry after a failed sync | built, tried on a real ESP32 board, the router unplugged ([tried.md](tried.md)) |
| Unit tests | 33 tests in Clock/test/test_clock.cpp, native build |
DOC
cat > docs/clock/tried.md <<'DOC'
# Clock — tried

## 2026-01-20, a real ESP32 board

Synced from pool.ntp.org, then the router unplugged for two minutes: the
clock retried every ten seconds and synced again four seconds after the
router came back.
DOC
cat > docs/clock/events.md <<'DOC'
---
sources: [Clock/include/ClockEvents.h]
checked: HEAD
---
# Clock events

The clock publishes two events on the bus:

- `clock/synced`, after each successful sync, with the offset corrected in
  milliseconds;
- `clock/sync-failed`, when a sync fails, with the number of failures in a
  row.
DOC
cat > docs/getting-started.md <<'DOC'
---
sources: [Clock/library.json, Clock/examples/basic/main.cpp]
checked: HEAD
---
# Getting started

Add the clock to `platformio.ini`:

```ini
lib_deps = clock@1.4.1
```

Then call `clock.begin()` in `setup()` and `clock.loop(millis())` in
`loop()`, as Clock/examples/basic/main.cpp does.
DOC
git add . && git commit -q -m "feat(clock): keep the time from NTP, drift corrected"
C=$(git rev-parse --short HEAD)
sed -i "s/checked: HEAD/checked: $C/" docs/clock/*.md docs/getting-started.md
git add . && git commit -q -m "docs: confirm the clock docs against the code"

# A sync every five minutes; the comment above the default is left as it was.
at 2026-02-01
sed -i 's/syncIntervalMs = 3600000;/syncIntervalMs = 300000;/' Clock/include/ClockConfig.h
git commit -qam "feat(clock): resync every five minutes"
C=$(git rev-parse --short HEAD)
sed -i 's/| 3600000 | the clock resyncs once an hour by default |/| 300000 | the clock resyncs every five minutes by default |/' docs/clock/README.md
sed -i "s/^checked: .*/checked: $C/" docs/clock/README.md
git commit -qam "docs(clock): say five minutes"

# The web UI's version fixed, a wider drift table, the legacy style gone,
# four tests merged.
# A person moves `checked` on every doc without reading them again.
at 2026-03-01
clock_h 481 > Clock/include/Clock.h
webui_h 0 CLOCK_VERSION > Clock/include/ClockWebUI.h
tests_cpp 29 > Clock/test/test_clock.cpp
lines Clock/include/Clock.h 569
lines Clock/include/ClockWebUI.h 236
git commit -qam "fix(clock): the web UI shows the library's version"
at 2026-03-02
C=$(git rev-parse --short HEAD)
sed -i "s/^checked: .*/checked: $C/" docs/clock/*.md docs/getting-started.md
git commit -qam "docs: confirm the docs after the fix"
