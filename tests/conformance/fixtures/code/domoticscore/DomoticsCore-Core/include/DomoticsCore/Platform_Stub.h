#ifndef DOMOTICS_CORE_PLATFORM_STUB_H
#define DOMOTICS_CORE_PLATFORM_STUB_H

/**
 * @file Platform_Stub.h
 * @brief Stub platform utilities for unsupported platforms.
 * 
 * This file provides empty/default implementations for platforms
 * that are not fully supported (AVR, ARM, unknown).
 */

#include <string>
#include <algorithm>
#include <cstdlib>
#include <climits>
#include <cassert>  // advanceMillisForTest refuses to run without an installed override
#include <cstdint>
#include <cstring>
#include <cmath>
#include <memory>
#include <chrono>
#include <thread>

// Define NAN if not available
#ifndef NAN
#define NAN (0.0f/0.0f)
#endif

// Arduino String compatibility for stub platforms
class String {

private:
    std::string data;

public:
    String() = default;
    String(const char* str) : data(str) {}
    String(const std::string& str) : data(str) {}
    String(char c) : data(1, c) {}
    String(int value) : data(std::to_string(value)) {}
    String(unsigned int value) : data(std::to_string(value)) {}
    String(long value) : data(std::to_string(value)) {}
    String(unsigned long value) : data(std::to_string(value)) {}
    // TEST-7: both cores print two decimals by default; std::to_string printed six.
    String(float value) : String(value, 2) {}
    String(double value) : String(value, 2) {}
    String(float value, int precision) {
        char buf[32];
        snprintf(buf, sizeof(buf), "%.*f", precision, value);
        data = buf;
    }
    String(double value, int precision) {
        char buf[32];
        snprintf(buf, sizeof(buf), "%.*f", precision, value);
        data = buf;
    }
    String(uint32_t value, int format) {
        switch (format) {
            case 16: // HEX
                char hex_buf[16];
                snprintf(hex_buf, sizeof(hex_buf), "%08X", value);
                data = hex_buf;
                break;
            case 10: // DEC
            default:
                data = std::to_string(value);
                break;
        }
    }
    
    const char* c_str() const { return data.c_str(); }
    bool isEmpty() const { return data.empty(); }
    int length() const { return static_cast<int>(data.length()); }

    void reserve(int size) { data.reserve(size); }
    void clear() { data.clear(); }
    
    // TEST-7: Arduino's toInt()/toFloat() are atol()/atof() — a numeric prefix,
    // 0 on garbage, no exception — and its long is 32-bit, so saturate there.
    long toInt() const {
        const long v = std::strtol(data.c_str(), nullptr, 10);
        return v > INT32_MAX ? INT32_MAX : (v < INT32_MIN ? INT32_MIN : v);
    }
    float toFloat() const { return std::strtof(data.c_str(), nullptr); }
    void toLowerCase() {
        std::transform(data.begin(), data.end(), data.begin(), ::tolower);
    }
    void toUpperCase() {
        std::transform(data.begin(), data.end(), data.begin(), ::toupper);
    }
    
    char operator[](int index) const { return data[index]; }
    
    String& operator+=(const String& other) { data += other.data; return *this; }
    String& operator+=(char c) { data += c; return *this; }
    String operator+(const String& other) const { return String(data + other.data); }
    
    // Allow const char* + String
    friend String operator+(const char* lhs, const String& rhs) {
        return String(std::string(lhs) + rhs.data);
    }
    
    bool operator==(const String& other) const { return data == other.data; }
    bool operator!=(const String& other) const { return data != other.data; }
    bool operator<(const String& other) const { return data < other.data; }

    // String search methods
    int indexOf(const char* str, int fromIndex = 0) const {
        if (fromIndex < 0 || fromIndex >= (int)data.length()) return -1;
        size_t pos = data.find(str, fromIndex);
        return (pos == std::string::npos) ? -1 : static_cast<int>(pos);
    }
    int indexOf(const String& str, int fromIndex = 0) const {
        return indexOf(str.c_str(), fromIndex);
    }
    int indexOf(char c, int fromIndex = 0) const {
        if (fromIndex < 0 || fromIndex >= (int)data.length()) return -1;
        size_t pos = data.find(c, fromIndex);
        return (pos == std::string::npos) ? -1 : static_cast<int>(pos);
    }

    // Reverse search methods
    int lastIndexOf(char c) const {
        size_t pos = data.rfind(c);
        return (pos == std::string::npos) ? -1 : static_cast<int>(pos);
    }
    int lastIndexOf(char c, int fromIndex) const {
        if (fromIndex < 0) return -1;
        size_t searchFrom = (fromIndex >= (int)data.length()) ? std::string::npos : static_cast<size_t>(fromIndex);
        size_t pos = data.rfind(c, searchFrom);
        return (pos == std::string::npos) ? -1 : static_cast<int>(pos);
    }
    int lastIndexOf(const String& str) const {
        size_t pos = data.rfind(str.data);
        return (pos == std::string::npos) ? -1 : static_cast<int>(pos);
    }
    int lastIndexOf(const String& str, int fromIndex) const {
        if (fromIndex < 0) return -1;
        size_t searchFrom = (fromIndex >= (int)data.length()) ? std::string::npos : static_cast<size_t>(fromIndex);
        size_t pos = data.rfind(str.data, searchFrom);
        return (pos == std::string::npos) ? -1 : static_cast<int>(pos);
    }

    bool startsWith(const String& prefix) const {
        return data.compare(0, prefix.data.length(), prefix.data) == 0;
    }
    bool endsWith(const String& suffix) const {
        if (suffix.data.length() > data.length()) return false;
        return data.compare(data.length() - suffix.data.length(), suffix.data.length(), suffix.data) == 0;
    }
    bool equalsIgnoreCase(const String& other) const {
        String a = *this; a.toLowerCase();
        String b = other; b.toLowerCase();
        return a.data == b.data;
    }

    String substring(int from) const {
        if (from < 0 || from >= (int)data.length()) return String();
        return String(data.substr(from));
    }
    String substring(int from, int to) const {
        if (from < 0 || from >= (int)data.length()) return String();
        if (to < from) return String();
        return String(data.substr(from, to - from));
    }

    void replace(const String& find, const String& replace) {
        size_t pos = 0;
        while ((pos = data.find(find.data, pos)) != std::string::npos) {
            data.replace(pos, find.data.length(), replace.data);
            pos += replace.data.length();
        }
    }

    void trim() {
        // Trim whitespace from both ends
        size_t start = data.find_first_not_of(" \t\r\n");
        size_t end = data.find_last_not_of(" \t\r\n");
        if (start == std::string::npos) {
            data.clear();
        } else {
            data = data.substr(start, end - start + 1);
        }
    }

    char charAt(int index) const {
        if (index < 0 || index >= (int)data.length()) return '\0';
        return data[index];
    }

    void remove(int index, int count = -1) {
        if (index >= 0 && index < (int)data.length()) {
            if (count < 0) count = data.length() - index; // Remove to end if count < 0
            data.erase(index, count);
        }
    }

    // Direct access to internal std::string
    // Note: Removed implicit operator std::string() to avoid ArduinoJson 7 conflicts
    const std::string& toStdString() const { return data; }
    std::string& toStdString() { return data; }

    // Explicit comparison with const char* to avoid ambiguity
    bool operator==(const char* other) const { return data == other; }
    bool operator!=(const char* other) const { return data != other; }

    // Write methods for ArduinoJson compatibility
    size_t write(uint8_t c) {
        data += static_cast<char>(c);
        return 1;
    }
    size_t write(const uint8_t* s, size_t n) {
        data.append(reinterpret_cast<const char*>(s), n);
        return n;
    }

    // Read method for ArduinoJson stream parsing compatibility
    // Maintains a read position for sequential reading
    mutable size_t readPos = 0;

    int read() const {
        if (readPos >= data.size()) return -1;
        return static_cast<unsigned char>(data[readPos++]);
    }

    int peek() const {
        if (readPos >= data.size()) return -1;
        return static_cast<unsigned char>(data[readPos]);
    }

    int available() const {
        return static_cast<int>(data.size() - readPos);
    }

    void resetReadPosition() const {
        readPos = 0;
    }

    // Size method for ArduinoJson string adapter
    size_t size() const { return data.size(); }
};

// Basic Arduino types for stub platforms
using byte = uint8_t;
using word = uint16_t;

// Arduino constants
#define HIGH 1
#define LOW 0
#define INPUT 0
#define OUTPUT 1
#define INPUT_PULLUP 2
#define HEX 16
#define DEC 10

// Stub logging macros for native/unsupported platforms
#ifndef log_e
#define log_e(fmt, ...) ((void)0)
#define log_w(fmt, ...) ((void)0)
#define log_i(fmt, ...) ((void)0)
#define log_d(fmt, ...) ((void)0)
#define log_v(fmt, ...) ((void)0)
#endif

namespace DomoticsCore {
namespace HAL {
namespace Platform {

/**
 * @brief Initialize logging system for stub platforms (no-op)
 */
inline void initializeLogging(long baudrate = 115200) {
    (void)baudrate; // No logging on stub platforms
}

/**
 * @brief Check if logging system is ready for stub platforms
 */
inline bool isLoggerReady() {
    return false; // No logging on stub platforms
}

// ---------------------------------------------------------------------------
// Scriptable clock (TEST-4).
//
// The WiFi component's remaining untested paths are all timer-gated — a 15 s
// connection timeout, a 5 s retry, a 30 s STA fallback — and the native clock
// was the wall clock, so a host test of the timeout would have taken 15 real
// seconds. The override is off by default: every suite that does not opt in
// sees the same steady_clock this file always returned.
//
// Install the override BEFORE constructing the component under test.
// NonBlockingDelay captures getMillis() at construction; a timer anchored to
// the wall clock and then read against an override at 0 wraps and reports
// ready once before re-anchoring — one spurious tick, enough to corrupt a
// scenario.
// ---------------------------------------------------------------------------
inline bool millisOverriddenForTest = false;
inline unsigned long overriddenMillisForTest = 0;

inline void setMillisForTest(unsigned long ms) {
    millisOverriddenForTest = true;
    overriddenMillisForTest = ms;
}
inline void advanceMillisForTest(unsigned long ms) {
    // Advancing a clock that is not installed would silently leave the wall
    // clock in charge — fail loudly instead.
    assert(millisOverriddenForTest && "setMillisForTest() must be called before advanceMillisForTest()");
    overriddenMillisForTest += ms;
}
inline void resetMillisForTest() {
    millisOverriddenForTest = false;
    overriddenMillisForTest = 0;
}

/**
 * @brief Get current milliseconds for stub platforms (native time)
 */
inline unsigned long getMillis() {
    if (millisOverriddenForTest) return overriddenMillisForTest;
    auto now = std::chrono::steady_clock::now();
    auto duration = now.time_since_epoch();
    return static_cast<unsigned long>(std::chrono::duration_cast<std::chrono::milliseconds>(duration).count());
}

/**
 * @brief Delay execution for stub platforms (native sleep)
 */
inline void delayMs(unsigned long ms) {
    std::this_thread::sleep_for(std::chrono::milliseconds(ms));
}

/**
 * @brief Yield execution to allow background tasks (stub: thread yield)
 */
inline void yield() {
    std::this_thread::yield();
}

/**
 * @brief Format chip ID as hexadecimal string for stub platforms
 */
inline String formatChipIdHex() {
    return String("00000000"); // Stub chip ID
}

/**
 * @brief Convert string to uppercase for stub platforms
 */
inline String toUpperCase(const String& str) {
    // Use std::string for stub implementation
    std::string upper = str.c_str();
    std::transform(upper.begin(), upper.end(), upper.begin(), ::toupper);
    return String(upper.c_str());
}

/**
 * @brief Get substring of string for stub platforms
 */
inline String substring(const String& str, int start, int end = -1) {
    std::string s = str.c_str();
    if (end == -1) end = s.length();
    return String(s.substr(start, end - start).c_str());
}

/**
 * @brief Find index of character in string for stub platforms
 */
inline int indexOf(const String& str, char ch) {
    std::string s = str.c_str();
    return static_cast<int>(s.find(ch));
}

/**
 * @brief Check if string starts with prefix for stub platforms
 */
inline bool startsWith(const String& str, const String& prefix) {
    std::string s = str.c_str();
    std::string p = prefix.c_str();
    return s.rfind(p, 0) == 0;
}

/**
 * @brief Check if string ends with suffix for stub platforms
 */
inline bool endsWith(const String& str, const String& suffix) {
    std::string s = str.c_str();
    std::string suf = suffix.c_str();
    if (s.length() < suf.length()) return false;
    return s.compare(s.length() - suf.length(), suf.length(), suf) == 0;
}
inline String getChipModel() {
#if defined(__AVR__)
    return "ATmega";
#elif defined(__arm__) && defined(ARDUINO)
    return "ARM Cortex";
#else
    return "Unknown";
#endif
}

/**
 * @brief Get chip revision (stub - always 0)
 */
inline uint8_t getChipRevision() {
    return 0;
}

/** @brief Chip id a test sets; 0 unless it does. */
inline uint64_t stubbedChipIdForTest = 0;
inline void setChipIdForTest(uint64_t id) { stubbedChipIdForTest = id; }

inline uint64_t getChipId() {
    return stubbedChipIdForTest;
}

/**
 * @brief Fill a buffer with bytes (stub — NOT random).
 *
 * The host has no hardware RNG and no security context: WebUI.h, the only
 * caller, needs <ESPAsyncWebServer.h> and never compiles natively. This exists
 * so any host translation unit that references the symbol links; it fills a
 * fixed, visible pattern rather than pretending to be random.
 */
inline void getRandomBytes(void* buf, size_t len) {
    uint8_t* p = static_cast<uint8_t*>(buf);
    for (size_t i = 0; i < len; ++i) p[i] = static_cast<uint8_t>(0xA5 ^ (i & 0xFF));
}

// Scriptable heap (TEST-4). The fixed 65536 below was chosen to keep every
// low-heap guard closed; the WiFi fallback ladder (2000/2500/3500/6500/10000)
// is nothing but such guards, so none of its branches was reachable natively.
// Default unchanged.
inline uint32_t stubbedFreeHeapForTest = 65536;

inline void setFreeHeapForTest(uint32_t bytes) { stubbedFreeHeapForTest = bytes; }
inline void resetFreeHeapForTest() { stubbedFreeHeapForTest = 65536; }

/**
 * @brief Get free heap memory (stub - returns realistic value)
 * Returns 65536 to avoid triggering low-heap guards in components
 * like WifiComponent that gate functionality on heap thresholds.
 */
inline uint32_t getFreeHeap() {
    return stubbedFreeHeapForTest;
}

/**
 * @brief Get CPU frequency in MHz (stub)
 */
inline uint32_t getCpuFreqMHz() {
#if defined(__AVR__)
    return F_CPU / 1000000UL;
#else
    return 0;
#endif
}

// Flight recorder seams (OBS-3). Largest block is scripted separately from
// free heap so a fragmentation cliff can be staged; the read counter lets a
// test assert how many heap walks a tick performed.
inline uint32_t stubbedLargestFreeBlockForTest = 32768;
inline unsigned int largestFreeBlockReadsForTest = 0;
inline void setLargestFreeBlockForTest(uint32_t bytes) { stubbedLargestFreeBlockForTest = bytes; }
inline void resetLargestFreeBlockForTest() { stubbedLargestFreeBlockForTest = 32768; largestFreeBlockReadsForTest = 0; }
inline uint32_t getAllocatableFreeHeap() { return stubbedFreeHeapForTest; }
inline uint32_t getLargestFreeBlock() { ++largestFreeBlockReadsForTest; return stubbedLargestFreeBlockForTest; }
inline constexpr uint32_t heapCliffThresholdBytes() { return 4096; }
inline constexpr uint8_t platformId() { return 0; }

// OBS-3: scripted failure of the restart-hook registration (the ESP32's full shutdown table).
inline bool restartHookInstallFailsForTest = false;

// Crash commands (OBS-3): the stub records the kind and dies of nothing.
inline char crashKindForTest[16] = {};
inline bool crashForTest(const char* kind) {
    static const char* known[] = { "abort", "oom", "null", "swdt", "hwdt", "hang", "restart", "nothrow", "squeeze", "release" };
    for (const char* k : known) {
        if (strcmp(k, kind) == 0) { snprintf(crashKindForTest, sizeof(crashKindForTest), "%s", kind); return true; }
    }
    return false;
}

// RTC memory: a RAM array that survives resetForTest() the way RTC survives
// a reset, and is cleared only by clearRtcForTest() — the power-cycle.
inline uint32_t stubRtcWordsForTest[96] = {};
inline constexpr uint32_t rtcWordsAvailable() { return 96; }
inline void clearRtcForTest() { for (auto& w : stubRtcWordsForTest) w = 0; }
inline bool rtcRead(uint32_t wordOffset, uint32_t* dst, size_t words) {
    if (wordOffset + words > 96) return false;
    for (size_t i = 0; i < words; ++i) dst[i] = stubRtcWordsForTest[wordOffset + i];
    return true;
}
inline bool rtcWrite(uint32_t wordOffset, const uint32_t* src, size_t words) {
    if (wordOffset + words > 96) return false;
    for (size_t i = 0; i < words; ++i) stubRtcWordsForTest[wordOffset + i] = src[i];
    return true;
}
inline uint32_t lastRtcStoreOffsetForTest = 0xFFFFFFFFu;   // OBS-4: which word a store sequence ended on
inline void rtcStoreWord(uint32_t wordOffset, uint32_t value) {
    if (wordOffset < 96) stubRtcWordsForTest[wordOffset] = value;
    lastRtcStoreOffsetForTest = wordOffset;
}

// OBS-4 seams: the heap hook the recorder installs, fired by a test; the
// ESP8266 globals scripted, cleared by the take as the platform does.
typedef void (*FailedAllocHook)(uint32_t size, uint32_t site);
inline FailedAllocHook failedAllocHookForTest = nullptr;
inline bool failedAllocHookInstallFailsForTest = false;
inline constexpr bool supportsFailedAllocHook() { return true; }
// fireOnInstall: a failure on another task at the moment of registration —
// the recorder must have finished writing RTC by then, or the group is erased.
inline bool failedAllocFireOnInstallForTest = false;
inline bool installFailedAllocHook(FailedAllocHook hook) {
    if (failedAllocHookInstallFailsForTest) { failedAllocHookForTest = nullptr; return false; }
    failedAllocHookForTest = hook;
    if (failedAllocFireOnInstallForTest && hook) hook(4096, 0x1800);
    return true;
}
inline void fireFailedAllocForTest(uint32_t size, uint32_t caps) { if (failedAllocHookForTest) failedAllocHookForTest(size, caps); }
inline uint32_t stubbedLastFailAllocAddrForTest = 0;
inline uint32_t stubbedLastFailAllocSizeForTest = 0;
inline void setLastFailedAllocForTest(uint32_t addr, uint32_t size) { stubbedLastFailAllocAddrForTest = addr; stubbedLastFailAllocSizeForTest = size; }
inline bool takeLastFailedAlloc(uint32_t& addr, uint32_t& size) {
    if (!stubbedLastFailAllocAddrForTest) return false;
    addr = stubbedLastFailAllocAddrForTest;
    size = stubbedLastFailAllocSizeForTest;
    stubbedLastFailAllocAddrForTest = stubbedLastFailAllocSizeForTest = 0;
    return true;
}
inline void failGroupEnter() {}
inline void failGroupLeave() {}
// A host build has one thread of its own and no interrupts to guard against.
inline uint32_t enterCoreLogCritical() { return 0; }
inline void leaveCoreLogCritical(uint32_t /*state*/) {}

inline uint32_t getMillisAnyContext() { return static_cast<uint32_t>(getMillis()); }
inline void resetFailedAllocForTest() {
    failedAllocHookForTest = nullptr;
    failedAllocHookInstallFailsForTest = false;
    failedAllocFireOnInstallForTest = false;
    stubbedLastFailAllocAddrForTest = stubbedLastFailAllocSizeForTest = 0;
    lastRtcStoreOffsetForTest = 0xFFFFFFFFu;
}

// Restart observability (TEST-4). The native body is a no-op, which made the
// WiFi reboot-to-STA path — whose only effect is this call — unassertable. The
// count is per-call, not latched: a pending reboot whose timer keeps firing
// increments it on every fire.
inline unsigned int restartCountForTest = 0;
inline void resetRestartCountForTest() { restartCountForTest = 0; }

/**
 * @brief Software reset (stub)
 */
// OBS-3: one hook the recorder installs so a restart the firmware asks for is
// marked as its own; defined in FlightRecorder.cpp.
typedef void (*RestartHook)();
bool installRestartHook(RestartHook hook);
RestartHook restartHook();
inline void restart() {
    if (restartHook()) restartHook()();
    ++restartCountForTest;
#if defined(__AVR__)
    asm volatile ("jmp 0");
#endif
}

/**
 * @brief Get chip temperature (stub)
 * @return NAN (temperature sensor not available)
 */
inline float getTemperature() {
    return NAN;
}

// =============================================================================
// Stub-Specific: System Information (Extended)
// =============================================================================

/**
 * @brief Get total heap size (stub - always 0)
 */
inline uint32_t getTotalHeap() {
    return 0;
}

/**
 * @brief Get minimum free heap ever recorded (stub - always 0)
 */
inline uint32_t getMinFreeHeap() {
    return 0;
}

/**
 * @brief Get maximum allocatable block size (stub - always 0)
 */
inline uint32_t getMaxAllocHeap() {
    return 0;
}

/**
 * @brief Get flash chip size (stub - always 0)
 */
inline uint32_t getFlashSize() {
    return 0;
}

/**
 * @brief Get sketch (program) size (stub - always 0)
 */
inline uint32_t getSketchSize() {
    return 0;
}

/**
 * @brief Get free sketch space (stub - always 0)
 */
inline uint32_t getFreeSketchSpace() {
    return 0;
}

// =============================================================================
// Stub-Specific: Reset Reason
// =============================================================================

/**
 * @brief Reset reason codes (platform-agnostic)
 */
enum class ResetReason : uint8_t {
    Unknown = 0,
    PowerOn = 1,
    External = 2,
    Software = 3,
    Panic = 4,
    IntWatchdog = 5,
    TaskWatchdog = 6,
    Watchdog = 7,
    DeepSleep = 8,
    Brownout = 9,
    SDIO = 10
};

/**
 * @brief Get reset reason for stub platform
 */
inline ResetReason resetReasonForTest = ResetReason::Unknown;   // scripted by setResetReasonForTest()

inline ResetReason getResetReason() {
    return resetReasonForTest;
}

/**
 * @brief Get human-readable reset reason string
 */
inline String getResetReasonString(ResetReason reason) {
    switch (reason) {
        case ResetReason::PowerOn:      return "Power-on";
        case ResetReason::External:     return "External reset";
        case ResetReason::Software:     return "Software reset";
        case ResetReason::Panic:        return "Panic/Exception";
        case ResetReason::IntWatchdog:  return "Interrupt watchdog";
        case ResetReason::TaskWatchdog: return "Task watchdog";
        case ResetReason::Watchdog:     return "Other watchdog";
        case ResetReason::DeepSleep:    return "Deep sleep wake";
        case ResetReason::Brownout:     return "Brownout";
        case ResetReason::SDIO:         return "SDIO reset";
        default:                        return "Unknown";
    }
}

/**
 * @brief Check if reset reason indicates an unexpected/crash reset
 */
inline bool wasUnexpectedReset(ResetReason reason) {
    return reason == ResetReason::Panic ||
           reason == ResetReason::IntWatchdog ||
           reason == ResetReason::TaskWatchdog ||
           reason == ResetReason::Watchdog ||
           reason == ResetReason::Brownout;
}

// =============================================================================
// Post-mortem diagnostics (OBS-1, OBS-2, OBS-7)
// =============================================================================

/**
 * @brief Exception registers the SDK preserved across the last reset.
 *
 * ESP8266 only. `valid` is true when the reset was an exception or a
 * watchdog (SDK reasons 1-3); `epc1` then locates the fault, or the loop the
 * watchdog interrupted, through `xtensa-lx106-elf-addr2line -e firmware.elf`.
 * An abort(), an assert and an out-of-memory `new` reach the next boot as
 * ResetReason::Software with nothing here — measured 2026-09-05 (OBS-2).
 */
struct ResetDetail {
    uint32_t exccause = 0, epc1 = 0, epc2 = 0, epc3 = 0, excvaddr = 0, depc = 0;
    bool valid = false;
};

/** @brief What the ESP32 core dump partition holds; unsupported elsewhere (OBS-1). */
struct CoreDumpStatus {
    bool supported = false;        // the platform can hold a core dump at all
    bool partitionPresent = false; // a `coredump` partition exists in the table
    bool dumpPresent = false;      // a dump from a previous panic is waiting in it
    uint32_t size = 0;             // its size in bytes
};

// Test seams, the TEST-4 pattern: scripted from the suite, byte-identical defaults.
inline ResetDetail resetDetailForTest{};
inline CoreDumpStatus coreDumpStatusForTest{};
inline uint32_t loopWatchdogSecondsForTest = 0;   // last value enableLoopWatchdog() accepted, 0 = never
inline uint32_t loopWatchdogFeedsForTest = 0;
inline String resetReasonCaveatForTest;           // scripted; empty = no caveat, the ESP32 shape
inline bool minFreeHeapTrackedForTest = false;    // scripted; false = the ESP8266 shape
inline bool loopWatchdogArmFailsForTest = false;  // scripted; true = the platform refuses to arm
inline uint8_t coreDumpImageForTest[256];         // scripted image bytes (OBS-1)
inline size_t coreDumpImageLenForTest = 0;
inline uint32_t coreDumpEraseCallsForTest = 0;

inline void setResetReasonForTest(ResetReason r) { resetReasonForTest = r; }
inline void setResetDetailForTest(const ResetDetail& d) { resetDetailForTest = d; }
inline void setCoreDumpStatusForTest(const CoreDumpStatus& s) { coreDumpStatusForTest = s; }
inline void resetDiagnosticsForTest() {
    resetReasonForTest = ResetReason::Unknown;
    resetDetailForTest = ResetDetail{};
    coreDumpStatusForTest = CoreDumpStatus{};
    loopWatchdogSecondsForTest = 0;
    loopWatchdogFeedsForTest = 0;
    resetReasonCaveatForTest = String();
    minFreeHeapTrackedForTest = false;
    loopWatchdogArmFailsForTest = false;
    coreDumpImageLenForTest = 0;
    coreDumpEraseCallsForTest = 0;
}
/** @brief Script an image: status says a dump of this size waits, reads return these bytes. */
inline void setCoreDumpImageForTest(const uint8_t* bytes, size_t len) {
    if (!bytes) len = 0;
    if (len > sizeof(coreDumpImageForTest)) len = sizeof(coreDumpImageForTest);   // a test asking for more gets the buffer's size back in status
    if (len) memcpy(coreDumpImageForTest, bytes, len);
    coreDumpImageLenForTest = len;
    coreDumpStatusForTest.supported = true;
    coreDumpStatusForTest.partitionPresent = true;
    coreDumpStatusForTest.dumpPresent = len > 0;
    coreDumpStatusForTest.size = static_cast<uint32_t>(len);
}

inline ResetDetail getResetDetail() { return resetDetailForTest; }
inline String getResetReasonCaveat(ResetReason /*reason*/) { return resetReasonCaveatForTest; }
inline bool tracksMinFreeHeap() { return minFreeHeapTrackedForTest; }
inline CoreDumpStatus getCoreDumpStatus() { return coreDumpStatusForTest; }
inline size_t coreDumpRead(uint32_t offset, uint8_t* buf, size_t len) {
    if (!buf || len == 0 || !coreDumpStatusForTest.dumpPresent || offset >= coreDumpImageLenForTest) return 0;
    if (len > coreDumpImageLenForTest - offset) len = coreDumpImageLenForTest - offset;
    memcpy(buf, coreDumpImageForTest + offset, len);
    return len;
}
inline bool coreDumpErase() {
    ++coreDumpEraseCallsForTest;
    if (!coreDumpStatusForTest.dumpPresent) return false;
    coreDumpStatusForTest.dumpPresent = false;
    coreDumpStatusForTest.size = 0;
    coreDumpImageLenForTest = 0;
    return true;
}
inline constexpr bool supportsLoopWatchdog() { return true; }   // mirrors ESP32 so the not-armed path is testable
inline bool enableLoopWatchdog(uint32_t seconds) {
    if (seconds == 0 || loopWatchdogArmFailsForTest) return false;
    loopWatchdogSecondsForTest = seconds;
    return true;
}
inline void feedLoopWatchdog() { if (loopWatchdogSecondsForTest) loopWatchdogFeedsForTest++; }

// =============================================================================
// Stub-Specific: LED Polarity
// =============================================================================

/**
 * @brief Get the correct value to turn LED_BUILTIN ON for stub platform
 * @return HIGH (default active-high)
 */
inline int ledBuiltinOn() {
    return HIGH;  // Default: active-high
}

/**
 * @brief Get the correct value to turn LED_BUILTIN OFF for stub platform
 * @return LOW (default active-high)
 */
inline int ledBuiltinOff() {
    return LOW;  // Default: active-high
}

/**
 * @brief Write digital value to pin (stub implementation)
 */
inline void digitalWrite(uint8_t pin, uint8_t value) {
    // Stub implementation - no actual GPIO on native platform
    (void)pin;
    (void)value;
}

/**
 * @brief Set pin mode (stub implementation)
 */
inline void pinMode(uint8_t pin, uint8_t mode) {
    // Stub implementation - no actual GPIO on native platform
    (void)pin;
    (void)mode;
}

/**
 * @brief Write analog/PWM value to pin (stub implementation)
 */
inline void analogWrite(uint8_t pin, int value) {
    // Stub implementation - no actual PWM on native platform
    (void)pin;
    (void)value;
}

/**
 * @brief Read digital value from pin (stub implementation)
 */
inline int digitalRead(uint8_t pin) {
    // Stub implementation - no actual GPIO on native platform
    (void)pin;
    return LOW;
}

/**
 * @brief Map a number from one range to another (stub implementation)
 */
inline long map(long value, long fromLow, long fromHigh, long toLow, long toHigh) {
    return (value - fromLow) * (toHigh - toLow) / (fromHigh - fromLow) + toLow;
}

/**
 * @brief Constrain a value between min and max (template version for any type)
 */
template<typename T>
inline T constrain(T value, T min_val, T max_val) {
    return (value < min_val) ? min_val : ((value > max_val) ? max_val : value);
}

/**
 * @brief Mathematical constant PI for stub platform
 */
constexpr double PI = 3.141592653589793238462643383279502884197169399375105820974944592307816406286;

/**
 * @brief Built-in LED pin number for stub platform
 */
#define LED_BUILTIN 0

/**
 * @brief Check if internal LED (LED_BUILTIN) has inverted logic on stub platform
 * @return false (default active-high)
 */
inline bool isInternalLEDInverted() {
    return false;  // Default: active-high
}

/**
 * @brief SHA256 hash computation (stub - no-op).
 */
class SHA256 {
public:
    SHA256() {}
    void begin() {}
    void update(const uint8_t*, size_t) {}
    void finish(uint8_t* digest) { memset(digest, 0, 32); }
    void abort() {}
    
    static String toHex(const uint8_t* digest, size_t len = 32) {
        static const char* hex = "0123456789abcdef";
        String out;
        out.reserve(len * 2);
        for (size_t i = 0; i < len; ++i) {
            out += hex[(digest[i] >> 4) & 0x0F];
            out += hex[digest[i] & 0x0F];
        }
        return out;
    }
};


// Counted, not held: a native test cannot otherwise see a mutation take the lock.
inline size_t s_stubLocksTaken = 0;
inline size_t s_stubLockDepth = 0;
inline size_t s_stubLockMaxDepth = 0;

/** @brief A recursive lock, empty here: nothing runs beside the test. */
class RecursiveLock {
public:
    RecursiveLock() = default;
    RecursiveLock(const RecursiveLock&) = delete;
    RecursiveLock& operator=(const RecursiveLock&) = delete;

    void lock() {
        ++s_stubLocksTaken;
        if (++s_stubLockDepth > s_stubLockMaxDepth) s_stubLockMaxDepth = s_stubLockDepth;
    }
    void unlock() { if (s_stubLockDepth) --s_stubLockDepth; }
};

/**
 * @brief What this platform's allocator and String cost, for callers that size
 *        a buffer or a queue in bytes.
 *
 * The host, and any target nobody has measured: pessimistic on purpose.
 */
struct AllocatorShape {
    static constexpr size_t kDequeNodeBytes = 80;
    static constexpr size_t kBlockOverhead  = 24;
    static constexpr size_t kSsoChars       = 10;
    static constexpr bool   kMeasured       = false;
};

} // namespace Platform

/** @brief Whether a socket can be opened yet; scriptable for tests. */
inline bool& canOpenServerState() {
    static bool ready = true;
    return ready;
}
inline bool canOpenServer() { return canOpenServerState(); }
inline void setCanOpenServerForTest(bool ready) { canOpenServerState() = ready; }
} // namespace HAL
} // namespace DomoticsCore

#endif // DOMOTICS_CORE_PLATFORM_STUB_H
