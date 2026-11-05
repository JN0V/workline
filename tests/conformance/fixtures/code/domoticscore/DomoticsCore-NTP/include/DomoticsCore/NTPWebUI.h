#pragma once

#include "DomoticsCore/NTP.h"
#include "DomoticsCore/IWebUIProvider.h"
#include "DomoticsCore/WebUI.h"
#include <ArduinoJson.h>
#include <cstdint>

namespace DomoticsCore {
namespace Components {
namespace WebUI {

// ========== Constexpr Timezone Lookup (stored in flash, not RAM) ==========

/**
 * @brief Timezone entry for constexpr lookup table
 */
struct TimezoneLookupEntry {
    const char* posix;       // POSIX timezone string
    const char* friendly;    // User-friendly name
};

/**
 * @brief Constexpr timezone lookup table (stored in flash on ESP8266/ESP32)
 *
 * This replaces std::map<String, String> to save ~3-4KB RAM on ESP8266.
 * Linear search is acceptable for 29 entries (O(29) worst case).
 */
static constexpr TimezoneLookupEntry TIMEZONE_LOOKUP[] = {
    {"UTC0", "UTC"},
    {"WET0WEST,M3.5.0/1,M10.5.0", "London"},
    {"CET-1CEST,M3.5.0,M10.5.0/3", "Paris (CET)"},
    {"EET-2EEST,M3.5.0/3,M10.5.0/4", "Athens (EET)"},
    {"MSK-3", "Moscow"},
    {"EST5EDT,M3.2.0,M11.1.0", "New York"},
    {"CST6CDT,M3.2.0,M11.1.0", "Chicago"},
    {"MST7MDT,M3.2.0,M11.1.0", "Denver"},
    {"PST8PDT,M3.2.0,M11.1.0", "Los Angeles"},
    {"AKST9AKDT,M3.2.0,M11.1.0", "Anchorage"},
    {"HST10", "Honolulu"},
    {"<-03>3", "Sao Paulo"},
    {"CST-8", "Shanghai"},
    {"JST-9", "Tokyo"},
    {"KST-9", "Seoul"},
    {"IST-5:30", "India"},
    {"PKT-5", "Karachi"},
    {"<+07>-7", "Bangkok"},
    {"WIB-7", "Jakarta"},
    {"GST-4", "Dubai"},
    {"AEST-10AEDT,M10.1.0,M4.1.0/3", "Sydney"},
    {"ACST-9:30ACDT,M10.1.0,M4.1.0/3", "Adelaide"},
    {"AWST-8", "Perth"},
    {"NZST-12NZDT,M9.5.0,M4.1.0/3", "Auckland"},
    {"EAT-3", "Nairobi"},
    {"SAST-2", "Johannesburg"},
    {"WAT-1", "Lagos"},
    {"EET-2EEST,M3.5.5/0,M10.5.5/0", "Jerusalem"},
    {"<+03>-3", "Riyadh"},
};

static constexpr size_t TIMEZONE_LOOKUP_COUNT = sizeof(TIMEZONE_LOOKUP) / sizeof(TIMEZONE_LOOKUP[0]);

/**
 * @brief WebUI provider for NTP component
 *
 * Provides web interface for NTP configuration, time display, and statistics.
 *
 * UI Contexts:
 * - ntp_status: Header badge showing sync status
 * - ntp_dashboard: Dashboard card with current time
 * - ntp_settings: Settings card for configuration
 * - ntp_detail: Component detail with statistics
 */
class NTPWebUI : public CachingWebUIProvider {
public:
    /**
     * @brief Construct WebUI provider
     * @param ntp Pointer to NTP component (non-owning)
     */
    explicit NTPWebUI(NTPComponent* ntp) : ntp(ntp) {
        // State tracking uses LazyState helper - no manual initialization needed
        // States will be initialized on first hasDataChanged() call
    }

    /**
     * @brief Set callback for NTP configuration persistence (optional)
     */
    void setConfigSaveCallback(std::function<void(const NTPConfig&)> callback) {
        onConfigSaved = callback;
    }

    /**
     * @brief Initialize WebUI routes (call after WebUI component is ready)
     * Registers the /api/ntp/timezones endpoint for on-demand timezone loading
     */
    void init(WebUIComponent* webui) {
        if (!webui) return;

        // Register timezone options endpoint - streams from flash to save RAM
        webui->registerApiRoute("/api/ntp/timezones", HTTP_GET, [](AsyncWebServerRequest* request) {
            // Stream timezone options from flash-stored constexpr table
            // This avoids allocating 29 String pairs in RAM
            AsyncResponseStream *response = request->beginResponseStream("application/json");
            response->print("[");

            for (size_t i = 0; i < TIMEZONE_LOOKUP_COUNT; ++i) {
                if (i > 0) response->print(",");
                response->print("{\"value\":\"");
                response->print(TIMEZONE_LOOKUP[i].posix);
                response->print("\",\"label\":\"");
                response->print(TIMEZONE_LOOKUP[i].friendly);
                response->print("\"}");
            }

            response->print("]");
            request->send(response);
        });
    }

private:
    std::function<void(const NTPConfig&)> onConfigSaved; // callback for persistence

    /**
     * @brief Get friendly timezone name from POSIX string
     * Uses constexpr lookup table (O(n) linear search, n=29)
     */
    String getTimezoneFriendlyName(const String& posixTz) const {
        for (size_t i = 0; i < TIMEZONE_LOOKUP_COUNT; ++i) {
            if (posixTz == TIMEZONE_LOOKUP[i].posix) {
                return String(TIMEZONE_LOOKUP[i].friendly);
            }
        }
        return posixTz;  // Fallback to POSIX string if not found
    }

public:

    // ========== IWebUIProvider Interface ==========

    String getWebUIName() const override {
        return ntp ? ntp->metadata.name : String("NTP");
    }

    String getWebUIVersion() const override {
        return ntp ? ntp->metadata.version : String("1.0.2");
    }

protected:
    // CachingWebUIProvider: build contexts once, they're cached for subsequent requests
    // OPTIMIZED for ESP8266: reduced from 3 to 2 contexts
    void buildContexts(std::vector<WebUIContext>& contexts) override {
        if (!ntp) return;

        // Header info - compact time display (also serves dashboard needs)
        contexts.push_back(WebUIContext::headerInfo("ntp_time", "Time", "dc-clock")
            .withField(WebUIField("time", "Time", WebUIFieldType::Display, "--:--:--", "", true))
            .withField(WebUIField("date", "Date", WebUIFieldType::Display, "", "", true))
            .withRealTime(1000)
            .withAPI("/api/ntp/time")
            .withPriority(100));

        // Settings card - Use placeholder values; real values come from getWebUIData()
        WebUIField timezoneField("timezone", "Timezone", WebUIFieldType::Select, "UTC0");
        timezoneField.endpoint = "/api/ntp/timezones";  // Options loaded on-demand

        contexts.push_back(WebUIContext::settings("ntp_settings", "NTP Configuration")
            .withField(WebUIField("enabled", "Enable NTP Sync", WebUIFieldType::Boolean, "true"))
            .withField(WebUIField("servers", "NTP Servers", WebUIFieldType::Text, "pool.ntp.org"))
            .withField(WebUIField("sync_interval", "Sync Interval (seconds)", WebUIFieldType::Number, "3600"))
            .withField(timezoneField)
            .withAPI("/api/ntp/settings"));
    }

public:

    String getWebUIData(const String& contextId) override {
        if (!ntp) return "{}";

        JsonDocument doc;

        if (contextId == "ntp_time") {
            // Provide time and date for header info zone
            doc["time"] = ntp->isSynced() ? ntp->getFormattedTime("%H:%M:%S") : "--:--:--";
            doc["date"] = ntp->isSynced() ? ntp->getFormattedTime("%Y-%m-%d") : "";

        } else if (contextId == "ntp_settings") {
            const NTPConfig& cfg = ntp->getConfig();
            doc["enabled"] = cfg.enabled;

            // Servers as comma-separated
            String serversStr;
            for (size_t i = 0; i < cfg.servers.size(); i++) {
                if (i > 0) serversStr += ", ";
                serversStr += cfg.servers[i];
            }
            doc["servers"] = serversStr;
            doc["sync_interval"] = cfg.syncInterval;  // seconds, as stored
            doc["timezone"] = cfg.timezone;

        }

        return webUIContextData(doc);
    }

    String handleWebUIRequest(const String& contextId, const String& /*endpoint*/,
                              const String& method, const std::map<String, String>& params) override {
        DLOG_D(LOG_NTP, "[WebUI] handleWebUIRequest: contextId=%s, method=%s", contextId.c_str(), method.c_str());
        
        if (!ntp) {
            DLOG_E(LOG_NTP, "[WebUI] NTP component not available");
            return "{\"success\":false,\"error\":\"Component not available\"}";
        }

        if (method == "GET") {
            return "{\"success\":true}";
        }

        if (method != "POST") {
            DLOG_W(LOG_NTP, "[WebUI] Method not allowed: %s", method.c_str());
            return "{\"success\":false,\"error\":\"Method not allowed\"}";
        }

        // Handle settings updates
        if (contextId == "ntp_settings") {
            auto fieldIt = params.find("field");
            auto valueIt = params.find("value");

            if (fieldIt != params.end()) {
                DLOG_D(LOG_NTP, "[WebUI] Field: %s", fieldIt->second.c_str());
            }
            if (valueIt != params.end()) {
                DLOG_D(LOG_NTP, "[WebUI] Value: %s", valueIt->second.c_str());
            }

            NTPConfig cfg = ntp->getConfig();

            // Handle field updates
            if (fieldIt != params.end() && valueIt != params.end()) {
                const String& field = fieldIt->second;
                const String& value = valueIt->second;

                // Handle configuration field updates
                if (field == "enabled") {
                    cfg.enabled = (value == "true" || value == "1");
                } else if (field == "servers") {
                    // Parse comma-separated servers
                    cfg.servers.clear();
                    String servers = value;
                    servers.trim();
                    int start = 0;
                    int commaPos;
                    while ((commaPos = servers.indexOf(',', start)) != -1) {
                        String server = servers.substring(start, commaPos);
                        server.trim();
                        if (server.length() > 0) {
                            cfg.servers.push_back(server);
                        }
                        start = commaPos + 1;
                    }
                    // Last server
                    String server = servers.substring(start);
                    server.trim();
                    if (server.length() > 0) {
                        cfg.servers.push_back(server);
                    }
                } else if (field == "sync_interval") {
                    // Seconds, an hour at least, and low enough that begin() can
                    // hand the interval to the SNTP client as milliseconds in a
                    // uint32_t — which caps it at 4 294 967 s, not at 2^32.
                    uint32_t seconds = 0;
                    if (!webUIParseUnsigned(value, seconds) || seconds < 3600u ||
                        seconds > (UINT32_MAX / 1000u)) {
                        DLOG_W(LOG_NTP, "[WebUI] invalid sync interval '%s'", value.c_str());
                        return "{\"success\":false,\"error\":\"Interval must be 3600-4294967 seconds\"}";
                    }
                    cfg.syncInterval = seconds;
                } else if (field == "timezone") {
                    cfg.timezone = value;
                } else {
                    // Unknown field
                    DLOG_W(LOG_NTP, "[WebUI] Unknown field: %s", field.c_str());
                    return "{\"success\":false,\"error\":\"Unknown field\"}";
                }

                ntp->setConfig(cfg);

                // Invoke persistence callback if set
                if (onConfigSaved) {
                    DLOG_I(LOG_NTP, "[WebUI] Invoking config save callback");
                    onConfigSaved(cfg);
                }

                // Trigger immediate sync after configuration change
                if (cfg.enabled) {
                    DLOG_I(LOG_NTP, "[WebUI] Triggering immediate sync after config save");
                    ntp->syncNow();
                }

                return "{\"success\":true}";
            }
        }

        return "{\"success\":false,\"error\":\"Unknown request\"}";
    }

    bool hasDataChanged(const String& contextId) override {
        if (!ntp) return false;

        // Use LazyState helper for timing-independent change tracking
        if (contextId == "ntp_time") {
            return ntpTimeState.hasChanged(ntp->getUnixTime());
        }
        else if (contextId == "ntp_settings") {
            const NTPConfig& cfg = ntp->getConfig();
            NTPSettingsState current = {cfg.enabled, cfg.timezone};
            return ntpSettingsState.hasChanged(current);
        }
        return true;
    }

private:
    NTPComponent* ntp;  // Non-owning pointer

    // State tracking using LazyState helper
    LazyState<time_t> ntpTimeState;

    struct NTPSettingsState {
        bool enabled;
        String timezone;

        bool operator==(const NTPSettingsState& other) const {
            return enabled == other.enabled && timezone == other.timezone;
        }
        bool operator!=(const NTPSettingsState& other) const { return !(*this == other); }
    };
    LazyState<NTPSettingsState> ntpSettingsState;
};

} // namespace WebUI
} // namespace Components
} // namespace DomoticsCore
