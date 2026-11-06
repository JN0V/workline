#ifndef DOMOTICS_CORE_HAL_NTP_H
#define DOMOTICS_CORE_HAL_NTP_H

/**
 * @file NTP_HAL.h
 * @brief NTP (Network Time Protocol) Hardware Abstraction Layer.
 * 
 * Provides a unified interface for time synchronization across platforms:
 * - ESP32: Uses esp_sntp API
 * - ESP8266: Uses lwIP's sntp client
 * - Other platforms: Stub implementation
 */

#include "DomoticsCore/Platform_HAL.h"
#include <time.h>

namespace DomoticsCore {
namespace HAL {
namespace NTP {

// Forward declarations for common functions
inline bool isSynced();
inline time_t getTime();

} // namespace NTP
} // namespace HAL
} // namespace DomoticsCore

// ============================================================================
// Platform-specific Implementation Files
// ============================================================================

#if DOMOTICS_PLATFORM_ESP32
    #include "NTP_ESP32.h"
#elif DOMOTICS_PLATFORM_ESP8266
    #include "NTP_ESP8266.h"
#else
    #include "NTP_Stub.h"
#endif

// ============================================================================
// Backward-Compatible API (delegates to NTPImpl namespace)
// ============================================================================

namespace DomoticsCore {
namespace HAL {
namespace NTP {

inline void init(const char* server1, const char* server2 = nullptr, const char* server3 = nullptr) {
    NTPImpl::init(server1, server2, server3);
}

inline void setTimezone(const char* tz) {
    NTPImpl::setTimezone(tz);
}

inline void setSyncInterval(uint32_t intervalMs) {
    NTPImpl::setSyncInterval(intervalMs);
}

inline void stop() {
    NTPImpl::stop();
}

inline void forceSync() {
    NTPImpl::forceSync();
}

inline time_t getTime() {
    return NTPImpl::now();
}

inline bool isSynced() {
    return getTime() > 1577836800;  // 2020-01-01 00:00:00 UTC
}

inline bool getFormattedTime(const char* format, char* buffer, size_t bufferSize) {
    if (!isSynced()) {
        if (bufferSize > 0) buffer[0] = '\0';
        return false;
    }
    
    time_t now = getTime();
    struct tm timeinfo;
    localtime_r(&now, &timeinfo);
    strftime(buffer, bufferSize, format, &timeinfo);
    return true;
}

} // namespace NTP
} // namespace HAL
} // namespace DomoticsCore

#endif // DOMOTICS_CORE_HAL_NTP_H
