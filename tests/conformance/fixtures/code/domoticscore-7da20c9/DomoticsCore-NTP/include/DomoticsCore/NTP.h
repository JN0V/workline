#pragma once

/**
 * @file NTP.h
 * @brief Declares the DomoticsCore NTP component for network time synchronization.
 * 
 * Provides NTP time synchronization with timezone support, formatted time strings,
 * and uptime tracking. Uses HAL abstraction for multi-platform support.
 * 
 * @example DomoticsCore-NTP/examples/BasicNTP/src/main.cpp
 * @example DomoticsCore-NTP/examples/NTPWithWebUI/src/main.cpp
 */

#include "DomoticsCore/IComponent.h"
#include "DomoticsCore/Logger.h"
#include "DomoticsCore/Timer.h"
#include "DomoticsCore/NTPEvents.h"
#include "DomoticsCore/Platform_HAL.h"  // For millis/delay abstractions
#include "NTP_HAL.h"  // Hardware Abstraction Layer for NTP
#include <cstdint>
#include <time.h>
#include <sys/time.h>
#include <vector>
#include <functional>

namespace DomoticsCore {
namespace Components {

// ========== Configuration ==========

/**
 * @brief NTP configuration structure
 */
struct NTPConfig {
    bool enabled = true;
    std::vector<String> servers = {"pool.ntp.org", "time.google.com", "time.cloudflare.com"};
    uint32_t syncInterval = 3600;    // seconds (1 hour)
    String timezone = "UTC0";         // POSIX TZ string
    uint32_t timeoutMs = 5000;       // Connection timeout
};

/**
 * @brief Common timezone presets
 */
namespace Timezones {
    constexpr const char* UTC = "UTC0";
    constexpr const char* EST = "EST5EDT,M3.2.0,M11.1.0";              // US Eastern
    constexpr const char* CST = "CST6CDT,M3.2.0,M11.1.0";              // US Central
    constexpr const char* MST = "MST7MDT,M3.2.0,M11.1.0";              // US Mountain
    constexpr const char* PST = "PST8PDT,M3.2.0,M11.1.0";              // US Pacific
    constexpr const char* CET = "CET-1CEST,M3.5.0,M10.5.0/3";          // Central European
    constexpr const char* GMT = "GMT0";                                 // Greenwich Mean Time
    constexpr const char* JST = "JST-9";                                // Japan
    constexpr const char* AEST = "AEST-10AEDT,M10.1.0,M4.1.0/3";      // Australia Eastern
    constexpr const char* IST = "IST-5:30";                            // India
    constexpr const char* NZST = "NZST-12NZDT,M9.5.0,M4.1.0/3";       // New Zealand
}

// ========== Statistics ==========

/**
 * @brief NTP synchronization statistics
 */
struct NTPStatistics {
    uint32_t syncCount = 0;
    uint32_t syncErrors = 0;
    time_t lastSyncTime = 0;
    uint32_t lastSyncDuration = 0;    // milliseconds
    time_t lastFailTime = 0;
    uint32_t consecutiveFailures = 0;
};

// ========== Component ==========

/**
 * @brief Network Time Protocol component
 * 
 * Provides NTP time synchronization with timezone support, formatted time strings,
 * and uptime tracking. Uses HAL abstraction for multi-platform support.
 * 
 * Features:
 * - Multiple NTP servers with automatic fallback
 * - Timezone management with DST support
 * - Configurable sync interval
 * - Sync status callbacks
 * - Formatted time strings (strftime)
 * - Uptime tracking
 * - Configuration persistence
 * 
 * @example Basic usage:
 * ```cpp
 * NTPConfig cfg;
 * cfg.timezone = Timezones::CET;
 * auto ntp = std::make_unique<NTPComponent>(cfg);
 * ntp->onSync([](bool success) {
 *     DLOG_I(LOG_NTP, "Time synced!");
 * });
 * core.addComponent(std::move(ntp));
 * ```
 */
class NTPComponent : public IComponent {
public:
    using SyncCallback = std::function<void(bool success)>;

    /**
     * @brief Construct NTP component with configuration
     * @param cfg NTP configuration
     */
    explicit NTPComponent(const NTPConfig& cfg = NTPConfig())
        : config(cfg), synced(false), syncInProgress(false), 
          bootTime(HAL::Platform::getMillis()), syncCallback(nullptr),
          syncTimeoutTimer(cfg.timeoutMs) {
        // Initialize component metadata immediately for dependency resolution
        metadata.name = "NTP";
        metadata.version = "1.4.3";  // x-release-please-version
        metadata.author = "DomoticsCore";
        metadata.description = "Network Time Protocol synchronization component";
        syncTimeoutTimer.disable();  // Start disabled
        DLOG_D(LOG_NTP, "Component constructed");
    }

    virtual ~NTPComponent() {
        if (clientStarted_) {
            HAL::NTP::stop();
        }
        DLOG_D(LOG_NTP, "Component destroyed");
    }

    // ========== IComponent Interface ==========

    ComponentStatus begin() override {
        DLOG_I(LOG_NTP, "Starting component...");
        begun_ = true;
        
        if (!config.enabled) {
            DLOG_W(LOG_NTP, "Component disabled");
            return ComponentStatus::Success;
        }

        // Set timezone via HAL
        HAL::NTP::setTimezone(config.timezone.c_str());
        DLOG_I(LOG_NTP, "Timezone set to: %s", config.timezone.c_str());

        for (size_t i = 0; i < config.servers.size() && i < 3; i++) {
            DLOG_I(LOG_NTP, "NTP server %zu: %s", i, config.servers[i].c_str());
        }
        if (config.servers.size() > 3) {
            DLOG_W(LOG_NTP, "%u NTP servers configured; the SNTP client takes three, the rest are ignored",
                   (unsigned)config.servers.size());
        }

        clientWanted_ = true;
        startClient();
        if (!clientStarted_) DLOG_I(LOG_NTP, "No network interface yet; SNTP client deferred");

        return ComponentStatus::Success;
    }

    ComponentStatus shutdown() override {
        begun_ = false;
        clientWanted_ = false;
        if (clientStarted_) {
            HAL::NTP::stop();
            DLOG_I(LOG_NTP, "SNTP client stopped via HAL");
        }
        clientStarted_ = false;
        return ComponentStatus::Success;
    }

    void loop() override {
        if (!config.enabled) return;
        if (clientWanted_ && !clientStarted_) startClient();

        // Check if time has been synced. Delegates to the HAL so the threshold
        // lives in exactly one place.
        time_t now = HAL::NTP::getTime();
        bool currentlySynced = HAL::NTP::isSynced();
        
        // Detect sync completion (initial or subsequent)
        if (currentlySynced) {
            if (!synced) {
                // First sync ever
                synced = true;
                stats.syncCount++;
                stats.lastSyncTime = now;
                stats.consecutiveFailures = 0;
                
                if (syncInProgress) {
                    stats.lastSyncDuration = syncTimeoutTimer.elapsed();
                    syncTimeoutTimer.disable();
                    syncInProgress = false;
                    DLOG_I(LOG_NTP, "Initial time sync completed after %u ms: %s", stats.lastSyncDuration, getFormattedTime().c_str());
                } else {
                    DLOG_I(LOG_NTP, "Initial time sync completed: %s", getFormattedTime().c_str());
                }
                
                // Emit event for orchestration
                emit(NTPEvents::EVENT_SYNCED, true);
                
                if (syncCallback) {
                    syncCallback(true);
                }
            } else if (syncInProgress && now != stats.lastSyncTime) {
                // Subsequent sync (manual or automatic)
                stats.syncCount++;
                stats.lastSyncTime = now;
                stats.lastSyncDuration = syncTimeoutTimer.elapsed();
                stats.consecutiveFailures = 0;
                syncTimeoutTimer.disable();
                syncInProgress = false;
                
                DLOG_I(LOG_NTP, "Time re-synchronized after %u ms: %s", stats.lastSyncDuration, getFormattedTime().c_str());
                
                // Emit event for orchestration
                emit(NTPEvents::EVENT_SYNCED, true);
                
                if (syncCallback) {
                    syncCallback(true);
                }
            }
        }
        
        // Check for sync timeout using NonBlockingDelay timer
        if (syncInProgress && syncTimeoutTimer.isReady()) {
            syncInProgress = false;
            syncTimeoutTimer.disable();
            stats.syncErrors++;
            stats.consecutiveFailures++;
            stats.lastFailTime = HAL::NTP::getTime();
            
            DLOG_W(LOG_NTP, "Sync timeout after %lu ms (no response from NTP servers)", syncTimeoutTimer.getInterval());
            
            // Emit event for orchestration
            emit(NTPEvents::EVENT_SYNC_FAILED, true);
            
            if (syncCallback) {
                syncCallback(false);
            }
        }
    }


    // ========== Time Synchronization ==========

    /**
     * @brief Trigger immediate NTP sync
     * @return True if sync initiated
     */
    bool syncNow() {
        DLOG_I(LOG_NTP, "syncNow() called");
        
        if (!config.enabled) {
            DLOG_W(LOG_NTP, "Component disabled, cannot sync");
            return false;
        }

        if (!clientStarted_) {
            DLOG_W(LOG_NTP, "No network interface yet, cannot sync");
            return false;
        }
        
        if (syncInProgress) {
            DLOG_W(LOG_NTP, "Sync already in progress, ignoring request");
            return false;
        }

        DLOG_I(LOG_NTP, "Requesting immediate SNTP sync via HAL...");
        syncInProgress = true;
        
        // Use configured timeout for sync
        syncTimeoutTimer.setInterval(config.timeoutMs);
        syncTimeoutTimer.reset();
        syncTimeoutTimer.enable();
        
        // Request immediate sync via HAL (non-blocking)
        HAL::NTP::forceSync();
        
        DLOG_I(LOG_NTP, "SNTP sync requested, timeout: %u ms", config.timeoutMs);
        return true;
    }

    /**
     * @brief Check if time is synced
     * @return True if time has been synced at least once
     */
    bool isSynced() const {
        // Delegate to the HAL: the year-2020 threshold is defined there, once.
        return synced && HAL::NTP::isSynced();
    }

    /**
     * @brief Get last successful sync time
     * @return Unix timestamp of last sync
     */
    time_t getLastSyncTime() const {
        return stats.lastSyncTime;
    }

    /**
     * @brief Get seconds until next automatic sync
     * @return Seconds until next sync, 0 if sync disabled
     */
    uint32_t getNextSyncIn() const {
        if (!isSynced() || !config.enabled) return 0;
        
        time_t now = HAL::NTP::getTime();
        time_t elapsed = now - stats.lastSyncTime;
        
        if (elapsed >= (time_t)config.syncInterval) {
            return 0;
        }
        
        return config.syncInterval - elapsed;
    }

    // ========== Time Access ==========

    /**
     * @brief Get current Unix timestamp
     * @return Seconds since epoch (1970-01-01 00:00:00 UTC)
     */
    time_t getUnixTime() const {
        return HAL::NTP::getTime();
    }

    /**
     * @brief Get local time as struct tm
     * @return Local time structure
     */
    struct tm getLocalTime() const {
        time_t now = HAL::NTP::getTime();
        struct tm timeinfo;
        localtime_r(&now, &timeinfo);
        return timeinfo;
    }

    /**
     * @brief Get formatted time string
     * @param format strftime format string
     * @return Formatted time string
     * 
     * Common formats:
     * - "%Y-%m-%d %H:%M:%S" -> "2025-10-02 19:30:45"
     * - "%Y/%m/%d" -> "2025/10/02"
     * - "%H:%M" -> "19:30"
     * - "%A, %B %d, %Y" -> "Thursday, October 02, 2025"
     */
    String getFormattedTime(const char* format = "%Y-%m-%d %H:%M:%S") const {
        if (!isSynced()) {
            return "Not synced";
        }
        
        struct tm timeinfo = getLocalTime();
        char buffer[128];
        strftime(buffer, sizeof(buffer), format, &timeinfo);
        return String(buffer);
    }

    /**
     * @brief Get ISO 8601 formatted time string
     * @return Time string in format "2025-10-02T19:30:45+02:00"
     */
    String getISO8601() const {
        if (!isSynced()) {
            return "Not synced";
        }
        
        struct tm timeinfo = getLocalTime();
        char buffer[64];
        
        // Format date and time
        strftime(buffer, sizeof(buffer), "%Y-%m-%dT%H:%M:%S", &timeinfo);
        String result = String(buffer);
        
        // Add timezone offset; the sign is kept apart so -00:30 stays negative
        const int offset = getGMTOffset();
        const int magnitude = offset < 0 ? -offset : offset;
        char tzBuffer[12];
        snprintf(tzBuffer, sizeof(tzBuffer), "%c%02d:%02d", offset < 0 ? '-' : '+',
                 magnitude / 3600, (magnitude % 3600) / 60);
        result += tzBuffer;
        
        return result;
    }

    // ========== Uptime ==========

    /**
     * @brief Get milliseconds since boot
     * @return Uptime in milliseconds
     */
    /// @note On ESP32/ESP8266, millis() wraps at ~49.7 days (32-bit unsigned long).
    /// The subtraction `HAL::Platform::getMillis() - bootTime` handles a single wrap
    /// correctly due to unsigned arithmetic. On 64-bit native test platforms, unsigned
    /// long is 64-bit and does not wrap. Return type is uint64_t to accommodate both.
    uint64_t getUptimeMs() const {
        return (uint64_t)(HAL::Platform::getMillis() - bootTime);
    }

    /**
     * @brief Get formatted uptime string
     * @return Uptime string in format "2d 5h 32m 15s"
     */
    String getFormattedUptime() const {
        uint64_t ms = getUptimeMs();
        uint32_t seconds = ms / 1000;
        
        uint32_t days = seconds / 86400;
        seconds %= 86400;
        uint32_t hours = seconds / 3600;
        seconds %= 3600;
        uint32_t minutes = seconds / 60;
        seconds %= 60;
        
        String result;
        if (days > 0) result += String(days) + "d ";
        if (hours > 0 || days > 0) result += String(hours) + "h ";
        if (minutes > 0 || hours > 0 || days > 0) result += String(minutes) + "m ";
        result += String(seconds) + "s";
        
        return result;
    }

    // ========== Timezone Management ==========

    /**
     * @brief Set timezone using POSIX TZ string
     * @param tz POSIX timezone string (e.g., "CET-1CEST,M3.5.0,M10.5.0/3")
     */
    void setTimezone(const String& tz) {
        config.timezone = tz;
        HAL::NTP::setTimezone(tz.c_str());
        DLOG_I(LOG_NTP, "Timezone changed to: %s", tz.c_str());
    }

    /**
     * @brief Get current timezone string
     * @return POSIX timezone string
     */
    String getTimezone() const {
        return config.timezone;
    }

    /**
     * @brief Get GMT offset in seconds
     * @return Offset from GMT in seconds (positive for east, negative for west)
     */
    int getGMTOffset() const {
        time_t now = HAL::NTP::getTime();
        struct tm utc_tm;
        struct tm local_tm;
        gmtime_r(&now, &utc_tm);
        localtime_r(&now, &local_tm);
        // Both read as UTC wall clocks: mktime() would apply the zone again and
        // take the UTC fields for standard time, an hour out during DST.
        return (int)(civilSeconds(local_tm) - civilSeconds(utc_tm));
    }

    /**
     * @brief Check if currently in Daylight Saving Time
     * @return True if DST is active
     */
    bool isDST() const {
        struct tm timeinfo = getLocalTime();
        return timeinfo.tm_isdst > 0;
    }

    // ========== Configuration ==========

    /**
     * @brief Get current NTP configuration
     * @return Current NTPConfig
     */
    const NTPConfig& getConfig() const {
        return config;
    }

    /**
     * @brief Update configuration
     * @param cfg New configuration
     */
    void setConfig(const NTPConfig& cfg) {
        // Compute diffs BEFORE mutating config
        bool needsRestart = (cfg.enabled != config.enabled) ||
                            (cfg.servers != config.servers) ||
                            (cfg.syncInterval != config.syncInterval);
        bool tzChanged = (cfg.timezone != config.timezone);

        // Apply new config
        config = cfg;

        // Apply timezone immediately without restart
        if (tzChanged) {
            setTimezone(config.timezone);
        }

        // Only a component begun and not shut down is restarted, and a disabled
        // one is then left stopped by begin(); before begin() the config is simply
        // what begin() will read.
        if (needsRestart && begun_) {
            if (clientStarted_) HAL::NTP::stop();
            clientStarted_ = false;
            begin();
        }
    }

    // ========== Callbacks ==========

    /**
     * @brief Register callback for sync events
     * @param callback Function called after each sync attempt (success/failure)
     */
    void onSync(SyncCallback callback) {
        syncCallback = callback;
    }

    // ========== Statistics ==========

    /**
     * @brief Get synchronization statistics
     * @return Statistics structure
     */
    const NTPStatistics& getStatistics() const {
        return stats;
    }

private:
    /**
     * @brief Start the SNTP client, when the platform says a socket can be used.
     *
     * On ESP32 the client reaches lwIP, which does not exist before the first
     * network interface; a caller that asks too early is deferred to loop().
     */
    void startClient() {
        if (clientStarted_ || !HAL::canOpenServer()) return;

        const char* srv1 = config.servers.size() > 0 ? config.servers[0].c_str() : "pool.ntp.org";
        const char* srv2 = config.servers.size() > 1 ? config.servers[1].c_str() : nullptr;
        const char* srv3 = config.servers.size() > 2 ? config.servers[2].c_str() : nullptr;

        // The client takes milliseconds in a uint32_t, so an interval past
        // ~49.7 days is held at the ceiling rather than wrapped into a value
        // that would resync several times a second.
        const uint32_t intervalMs = (config.syncInterval > UINT32_MAX / 1000u)
                                        ? UINT32_MAX
                                        : config.syncInterval * 1000u;
        HAL::NTP::setSyncInterval(intervalMs);
        HAL::NTP::init(srv1, srv2, srv3);
        clientStarted_ = true;
        DLOG_I(LOG_NTP, "SNTP client started via HAL");
    }

    NTPConfig config;
    NTPStatistics stats;
    bool synced;
    bool syncInProgress;
    bool begun_ = false;          // between begin() and shutdown()
    bool clientWanted_ = false;   // begin() wants the client, shutdown() does not: loop() reads this
    bool clientStarted_ = false;
    /// @note On ESP32/ESP8266, unsigned long is 32-bit and wraps at ~49.7 days
    /// (inherent millis() limitation). On 64-bit native test platforms, unsigned long
    /// is 64-bit and does not wrap. This difference is acceptable for testing purposes
    /// but should be considered if writing tests that simulate long uptimes.
    unsigned long bootTime;
    SyncCallback syncCallback;
    // Seconds since 1970 of a broken-down time read as UTC (days from civil).
    static int64_t civilSeconds(const struct tm& t) {
        int y = t.tm_year + 1900;
        const unsigned m = (unsigned)t.tm_mon + 1;
        y -= m <= 2;
        const int era = (y >= 0 ? y : y - 399) / 400;
        const unsigned yoe = (unsigned)(y - era * 400);
        const unsigned doy = (153 * (m > 2 ? m - 3 : m + 9) + 2) / 5 + (unsigned)t.tm_mday - 1;
        const unsigned doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
        const int64_t days = (int64_t)era * 146097 + (int64_t)doe - 719468;
        return days * 86400 + t.tm_hour * 3600 + t.tm_min * 60 + t.tm_sec;
    }

    Utils::NonBlockingDelay syncTimeoutTimer;  // Timer for sync timeout tracking
};

} // namespace Components
} // namespace DomoticsCore
