#ifndef DOMOTICS_CORE_LOGGER_H
#define DOMOTICS_CORE_LOGGER_H

#include "Platform_HAL.h"
#include "CoreLog_HAL.h"   // the capture this framework's own output must not feed
#include <vector>
#include <functional>
#include <algorithm>

// Log levels enum
enum LogLevel {
    LOG_LEVEL_NONE = 0,
    LOG_LEVEL_ERROR = 1,
    LOG_LEVEL_WARN = 2,
    LOG_LEVEL_INFO = 3,
    LOG_LEVEL_DEBUG = 4,
    LOG_LEVEL_VERBOSE = 5
};

// Logger callback system for RemoteConsole and other listeners
class LoggerCallbacks {
public:
    using CallbackFn = std::function<void(LogLevel, const char*, const char*)>;
    using CallbackId = uint8_t;

    static CallbackId addCallback(CallbackFn cb) {
        CallbackId id = nextId()++;
        getEntries().push_back({id, std::move(cb)});
        return id;
    }

    static void removeCallback(CallbackId id) {
        auto& entries = getEntries();
        entries.erase(
            std::remove_if(entries.begin(), entries.end(),
                           [id](const Entry& e) { return e.id == id; }),
            entries.end());
        entries.shrink_to_fit();
    }

    static void broadcast(LogLevel level, const char* tag, const char* message) {
        for (auto& e : getEntries()) {
            if (e.cb) {
                e.cb(level, tag, message);
            }
        }
    }

private:
    struct Entry {
        CallbackId id;
        CallbackFn cb;
    };

    static uint8_t& nextId() {
        static uint8_t id = 0;
        return id;
    }

    // Use Meyer's singleton to avoid static initialization order issues
    static std::vector<Entry>& getEntries() {
        static std::vector<Entry> entries;
        return entries;
    }
};

// Unified logging macros — platform-agnostic
// DLOG_SNPRINTF and DOMOTICS_DLOG_BUF_SIZE are defined by HAL:
//   ESP8266: PROGMEM format strings + 128-byte buffer (saves ~11KB DRAM)
//   ESP32/Other: standard snprintf + 256-byte buffer

#define DLOG_E(tag, format, ...) \
    do { \
        char _log_buf[DOMOTICS_DLOG_BUF_SIZE]; \
        DLOG_SNPRINTF(_log_buf, sizeof(_log_buf), format, ##__VA_ARGS__); \
        { DOMOTICS_LOG_OWN_OUTPUT_GUARD log_e("[%s] %s", tag, _log_buf); } \
        LoggerCallbacks::broadcast(LOG_LEVEL_ERROR, tag, _log_buf); \
    } while(0)

#define DLOG_W(tag, format, ...) \
    do { \
        char _log_buf[DOMOTICS_DLOG_BUF_SIZE]; \
        DLOG_SNPRINTF(_log_buf, sizeof(_log_buf), format, ##__VA_ARGS__); \
        { DOMOTICS_LOG_OWN_OUTPUT_GUARD log_w("[%s] %s", tag, _log_buf); } \
        LoggerCallbacks::broadcast(LOG_LEVEL_WARN, tag, _log_buf); \
    } while(0)

#define DLOG_I(tag, format, ...) \
    do { \
        char _log_buf[DOMOTICS_DLOG_BUF_SIZE]; \
        DLOG_SNPRINTF(_log_buf, sizeof(_log_buf), format, ##__VA_ARGS__); \
        { DOMOTICS_LOG_OWN_OUTPUT_GUARD log_i("[%s] %s", tag, _log_buf); } \
        LoggerCallbacks::broadcast(LOG_LEVEL_INFO, tag, _log_buf); \
    } while(0)

#define DLOG_D(tag, format, ...) \
    do { \
        char _log_buf[DOMOTICS_DLOG_BUF_SIZE]; \
        DLOG_SNPRINTF(_log_buf, sizeof(_log_buf), format, ##__VA_ARGS__); \
        { DOMOTICS_LOG_OWN_OUTPUT_GUARD log_d("[%s] %s", tag, _log_buf); } \
        LoggerCallbacks::broadcast(LOG_LEVEL_DEBUG, tag, _log_buf); \
    } while(0)

#define DLOG_V(tag, format, ...) \
    do { \
        char _log_buf[DOMOTICS_DLOG_BUF_SIZE]; \
        DLOG_SNPRINTF(_log_buf, sizeof(_log_buf), format, ##__VA_ARGS__); \
        { DOMOTICS_LOG_OWN_OUTPUT_GUARD log_v("[%s] %s", tag, _log_buf); } \
        LoggerCallbacks::broadcast(LOG_LEVEL_VERBOSE, tag, _log_buf); \
    } while(0)

// Convenience macros for simple string messages (no formatting)
#define DLOG_ES(tag, message) DLOG_E(tag, "%s", message)
#define DLOG_WS(tag, message) DLOG_W(tag, "%s", message)
#define DLOG_IS(tag, message) DLOG_I(tag, "%s", message)
#define DLOG_DS(tag, message) DLOG_D(tag, "%s", message)
#define DLOG_VS(tag, message) DLOG_V(tag, "%s", message)

// Standard component tags (can be used by library and apps)
#define LOG_CORE     "CORE"
#define LOG_WIFI     "WIFI"
#define LOG_MQTT     "MQTT"
#define LOG_HTTP     "HTTP"
#define LOG_HA       "HA"
#define LOG_OTA      "OTA"
#define LOG_LED      "LED"
#define LOG_SECURITY "SECURITY"
#define LOG_WEB      "WEB"
#define LOG_SYSTEM   "SYSTEM"
#define LOG_STORAGE  "STORAGE"
#define LOG_NTP      "NTP"
#define LOG_CONSOLE  "CONSOLE"
// The lines the platform writes itself (the Arduino core, the SDK, ESP-IDF),
// captured rather than emitted: a tag of their own so a filter can separate them
// from this framework's CORE.
#define LOG_PLATFORM "PLATFORM"

// Log levels (for reference - controlled by CORE_DEBUG_LEVEL):
// 0 = None
// 1 = Error only
// 2 = Error + Warning
// 3 = Error + Warning + Info (default)
// 4 = Error + Warning + Info + Debug
// 5 = All (including Verbose)

#endif // DOMOTICS_CORE_LOGGER_H
