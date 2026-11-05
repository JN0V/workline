#ifndef DOMOTICS_CORE_HAL_PLATFORM_H
#define DOMOTICS_CORE_HAL_PLATFORM_H

/**
 * @file Platform_HAL.h
 * @brief Platform detection and compatibility macros for DomoticsCore.
 * 
 * This header provides compile-time platform detection and defines common
 * macros to enable platform-specific code paths throughout the library.
 * 
 * Supported Platforms:
 * - ESP32 (full support)
 * - ESP8266 (partial support - planned)
 * - Arduino AVR (very limited - LED only)
 * - Arduino ARM (limited support - planned)
 */

// ============================================================================
// Platform Detection
// ============================================================================

// ESP32 (primary platform)
#if defined(ESP32) || defined(ARDUINO_ARCH_ESP32)
    #define DOMOTICS_PLATFORM_ESP32 1
    #define DOMOTICS_PLATFORM_NAME "ESP32"
    #define DOMOTICS_HAS_WIFI 1
    #define DOMOTICS_HAS_PREFERENCES 1
    #define DOMOTICS_HAS_FREERTOS 1
    #define DOMOTICS_HAS_ASYNC_TCP 1
    #define DOMOTICS_HAS_SNTP 1
    #define DOMOTICS_HAS_OTA 1
    #define DOMOTICS_HAS_SPIFFS 1
    #define DOMOTICS_HAS_LITTLEFS 1
    #define DOMOTICS_RAM_SIZE_KB 320
    #define DOMOTICS_FLASH_SIZE_KB 4096

// ESP8266
#elif defined(ESP8266) || defined(ARDUINO_ARCH_ESP8266)
    #define DOMOTICS_PLATFORM_ESP8266 1
    #define DOMOTICS_PLATFORM_NAME "ESP8266"
    #define DOMOTICS_HAS_WIFI 1
    #define DOMOTICS_HAS_PREFERENCES 0  // Use EEPROM or LittleFS instead
    #define DOMOTICS_HAS_FREERTOS 0
    #define DOMOTICS_HAS_ASYNC_TCP 0    // Different library needed
    #define DOMOTICS_HAS_SNTP 0         // No esp_sntp: lwIP's sntp client instead
    #define DOMOTICS_HAS_OTA 1
    #define DOMOTICS_HAS_SPIFFS 1
    #define DOMOTICS_HAS_LITTLEFS 1
    #define DOMOTICS_RAM_SIZE_KB 80
    #define DOMOTICS_FLASH_SIZE_KB 4096

// Arduino AVR (Uno, Mega, etc.)
#elif defined(__AVR__)
    #define DOMOTICS_PLATFORM_AVR 1
    #define DOMOTICS_PLATFORM_NAME "AVR"
    #define DOMOTICS_HAS_WIFI 0
    #define DOMOTICS_HAS_PREFERENCES 0
    #define DOMOTICS_HAS_FREERTOS 0
    #define DOMOTICS_HAS_ASYNC_TCP 0
    #define DOMOTICS_HAS_SNTP 0
    #define DOMOTICS_HAS_OTA 0
    #define DOMOTICS_HAS_SPIFFS 0
    #define DOMOTICS_HAS_LITTLEFS 0
    #define DOMOTICS_RAM_SIZE_KB 2      // Uno has 2KB
    #define DOMOTICS_FLASH_SIZE_KB 32

// Arduino ARM (Due, SAMD, etc.)
#elif defined(__arm__) && defined(ARDUINO)
    #define DOMOTICS_PLATFORM_ARM 1
    #define DOMOTICS_PLATFORM_NAME "ARM"
    #define DOMOTICS_HAS_WIFI 0         // May be available with shield
    #define DOMOTICS_HAS_PREFERENCES 0
    #define DOMOTICS_HAS_FREERTOS 0
    #define DOMOTICS_HAS_ASYNC_TCP 0
    #define DOMOTICS_HAS_SNTP 0
    #define DOMOTICS_HAS_OTA 0
    #define DOMOTICS_HAS_SPIFFS 0
    #define DOMOTICS_HAS_LITTLEFS 0
    #define DOMOTICS_RAM_SIZE_KB 96     // Due has 96KB
    #define DOMOTICS_FLASH_SIZE_KB 512

// Unknown platform (native tests, etc.)
#else
    #define DOMOTICS_PLATFORM_UNKNOWN 1
    #define DOMOTICS_PLATFORM_NAME "Unknown"
    #define DOMOTICS_HAS_WIFI 0
    #define DOMOTICS_HAS_PREFERENCES 0
    #define DOMOTICS_HAS_FREERTOS 0
    #define DOMOTICS_HAS_ASYNC_TCP 0
    #define DOMOTICS_HAS_SNTP 0
    #define DOMOTICS_HAS_OTA 0
    #define DOMOTICS_HAS_SPIFFS 0
    #define DOMOTICS_HAS_LITTLEFS 0
    #define DOMOTICS_RAM_SIZE_KB 0
    #define DOMOTICS_FLASH_SIZE_KB 0
#endif

// ============================================================================
// Feature Availability Checks (compile-time)
// ============================================================================

/**
 * @brief Check if platform supports WiFi
 */
#define DOMOTICS_SUPPORTS_WIFI() (DOMOTICS_HAS_WIFI == 1)

/**
 * @brief Check if platform supports persistent storage
 */
#define DOMOTICS_SUPPORTS_STORAGE() (DOMOTICS_HAS_PREFERENCES == 1 || DOMOTICS_HAS_LITTLEFS == 1)

/**
 * @brief Check if platform supports network time
 */
#define DOMOTICS_SUPPORTS_NTP() (DOMOTICS_HAS_WIFI == 1)

/**
 * @brief Check if platform supports OTA updates
 */
#define DOMOTICS_SUPPORTS_OTA() (DOMOTICS_HAS_OTA == 1)

/**
 * @brief Check if platform has enough RAM for full framework
 * (EventBus, WebUI, JSON parsing require ~20KB+ RAM)
 */
#define DOMOTICS_SUPPORTS_FULL_FRAMEWORK() (DOMOTICS_RAM_SIZE_KB >= 80)

// ============================================================================
// Platform-specific Implementation Files
// ============================================================================

#if DOMOTICS_PLATFORM_ESP32
    #include "Platform_ESP32.h"
#elif DOMOTICS_PLATFORM_ESP8266
    #include "Platform_ESP8266.h"
#else
    #include "Platform_Stub.h"
#endif

#include <string.h>
namespace DomoticsCore { namespace HAL { namespace Platform {
/** @brief Holds a RecursiveLock for a scope. */
class LockGuard {
public:
    explicit LockGuard(RecursiveLock& lock) : lock_(lock) { lock_.lock(); }
    ~LockGuard() { lock_.unlock(); }
    LockGuard(const LockGuard&) = delete;
    LockGuard& operator=(const LockGuard&) = delete;

private:
    RecursiveLock& lock_;
};

/** @brief The crash kinds crashForTest() returns from (OBS-4): the console says "done:" for these. */
inline bool crashKindSurvives(const char* kind) {
    return strcmp(kind, "nothrow") == 0 || strcmp(kind, "squeeze") == 0 || strcmp(kind, "release") == 0;
}
}}}

// Default log buffer size if not defined by platform
#ifndef DOMOTICS_LOG_BUFFER_SIZE
    #define DOMOTICS_LOG_BUFFER_SIZE 100
#endif

// ============================================================================
// Cross-platform PROGMEM helpers
// On ESP8266, .rodata is in DRAM — use PSTR/PROGMEM to move to Flash.
// On ESP32 and other platforms, .rodata is already Flash-mapped — no-op.
// Defined in platform HAL files; fallbacks here for non-ESP8266.
// ============================================================================
#ifndef DSNPRINTF_P
    #define DSNPRINTF_P(buf, size, fmt, ...) \
        snprintf(buf, size, fmt, ##__VA_ARGS__)
#endif
#ifndef DLOG_SNPRINTF
    #define DLOG_SNPRINTF(buf, size, fmt, ...) \
        snprintf(buf, size, fmt, ##__VA_ARGS__)
#endif
#ifndef DOMOTICS_DLOG_BUF_SIZE
    #define DOMOTICS_DLOG_BUF_SIZE 256
#endif
// Platforms whose own logger writes through a sink the core-log capture holds
// define this to a guard, so a line this framework prints is not captured and
// shown twice. Everywhere else it costs nothing.
#ifndef DOMOTICS_LOG_OWN_OUTPUT_GUARD
    #define DOMOTICS_LOG_OWN_OUTPUT_GUARD
#endif
// The Arduino cores define these; nothing does on the host, so headers that
// place literals in flash could not be compiled by a native test at all.
#ifndef PROGMEM
    #define PROGMEM
#endif
#ifndef PSTR
    #define PSTR(s) (s)
#endif

// ============================================================================
// Backward-Compatible API (delegates to Platform namespace)
// ============================================================================

namespace DomoticsCore {
namespace HAL {

/**
 * @brief Initialize platform-specific logging system
 */
inline void initializeLogging(long baudrate = 115200) {
    Platform::initializeLogging(baudrate);
}

/**
 * @brief Check if logging system is ready
 */
inline bool isLoggerReady() {
    return Platform::isLoggerReady();
}

/**
 * @brief Get current milliseconds
 */
inline unsigned long getMillis() {
    return Platform::getMillis();
}

/**
 * @brief Delay execution
 */
inline void delay(unsigned long ms) {
    Platform::delayMs(ms);
}

/**
 * @brief Format chip ID as hexadecimal string
 */
inline String formatChipIdHex() {
    return Platform::formatChipIdHex();
}

/**
 * @brief Convert string to uppercase
 */
inline String toUpperCase(const String& str) {
    return Platform::toUpperCase(str);
}

/**
 * @brief Get substring of string
 */
inline String substring(const String& str, int start, int end = -1) {
    return Platform::substring(str, start, end);
}

/**
 * @brief Find index of character in string
 */
inline int indexOf(const String& str, char ch) {
    return Platform::indexOf(str, ch);
}

/**
 * @brief Check if string starts with prefix
 */
inline bool startsWith(const String& str, const String& prefix) {
    return Platform::startsWith(str, prefix);
}

/**
 * @brief Check if string ends with suffix
 */
inline bool endsWith(const String& str, const String& suffix) {
    return Platform::endsWith(str, suffix);
}

/**
 * @brief Get platform name as string
 */
inline const char* getPlatformName() {
    return DOMOTICS_PLATFORM_NAME;
}

/**
 * @brief Get chip model/ID (delegates to Platform namespace)
 */
inline String getChipModel() {
    return Platform::getChipModel();
}

/**
 * @brief Get chip revision (delegates to Platform namespace)
 */
inline uint8_t getChipRevision() {
    return Platform::getChipRevision();
}

/**
 * @brief Get unique chip ID (delegates to Platform namespace)
 */
inline uint64_t getChipId() {
    return Platform::getChipId();
}

/**
 * @brief Get free heap memory (delegates to Platform namespace)
 */
inline uint32_t getFreeHeap() {
    return Platform::getFreeHeap();
}

/**
 * @brief Get total RAM size in KB
 */
inline uint32_t getTotalRAM_KB() {
    return DOMOTICS_RAM_SIZE_KB;
}

/**
 * @brief Get CPU frequency in MHz (delegates to Platform namespace)
 */
inline uint32_t getCpuFreqMHz() {
    return Platform::getCpuFreqMHz();
}

/**
 * @brief Software reset (delegates to Platform namespace)
 */
inline void restart() {
    Platform::restart();
}

/**
 * @brief Get the correct value to turn LED_BUILTIN ON for the current platform
 * @return HIGH or LOW depending on platform LED polarity
 */
inline int ledBuiltinOn() {
    return Platform::ledBuiltinOn();
}

/**
 * @brief Get the correct value to turn LED_BUILTIN OFF for the current platform
 * @return HIGH or LOW depending on platform LED polarity
 */
inline int ledBuiltinOff() {
    return Platform::ledBuiltinOff();
}

/**
 * @brief Check if internal LED (LED_BUILTIN) has inverted logic
 * @return true if LED is active-low (ESP8266), false if active-high (ESP32)
 */
inline bool isInternalLEDInverted() {
    return Platform::isInternalLEDInverted();
}

/**
 * @brief Write digital value to pin (delegates to Platform namespace)
 */
inline void digitalWrite(uint8_t pin, uint8_t value) {
    Platform::digitalWrite(pin, value);
}

/**
 * @brief Set pin mode (delegates to Platform namespace)
 */
inline void pinMode(uint8_t pin, uint8_t mode) {
    Platform::pinMode(pin, mode);
}

/**
 * @brief Write analog/PWM value to pin (delegates to Platform namespace)
 */
inline void analogWrite(uint8_t pin, int value) {
    Platform::analogWrite(pin, value);
}

/**
 * @brief Read digital value from pin (delegates to Platform namespace)
 */
inline int digitalRead(uint8_t pin) {
    return Platform::digitalRead(pin);
}

/**
 * @brief Map a number from one range to another (delegates to Platform namespace)
 */
inline long map(long value, long fromLow, long fromHigh, long toLow, long toHigh) {
    return Platform::map(value, fromLow, fromHigh, toLow, toHigh);
}

/**
 * @brief SHA256 - use Platform::SHA256 directly or this alias
 */
using SHA256 = Platform::SHA256;

/**
 * @brief Mathematical constant PI for platform independence
 */
constexpr double PI = Platform::PI;

} // namespace HAL
} // namespace DomoticsCore

#endif // DOMOTICS_CORE_HAL_PLATFORM_H
