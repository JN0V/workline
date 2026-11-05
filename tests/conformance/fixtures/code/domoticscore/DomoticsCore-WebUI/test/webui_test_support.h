#pragma once

// Shared by the WebUI component suites: includes and the two test providers.

#include <unity.h>
#include <DomoticsCore/Core.h>
#include <DomoticsCore/IWebUIProvider.h>
#include <DomoticsCore/WebUI/WebUIConfig.h>
#include <DomoticsCore/WebUI/ProviderRegistry.h>
#include <DomoticsCore/WebUI/StreamingContextSerializer.h>
#include <DomoticsCore/Testing/HeapTracker.h>
#include <ArduinoJson.h>
using namespace DomoticsCore;
using namespace DomoticsCore::Components;
using namespace DomoticsCore::Components::WebUI;
using namespace DomoticsCore::Testing;

// ============================================================================
// Mock Provider using CachingWebUIProvider (NEW - memory optimized)
// ============================================================================

class MockWebUIProvider : public CachingWebUIProvider {
private:
    String name_;
    String version_;
    std::vector<WebUIContext> pendingContexts_;  // Contexts to add before first access
    bool enabled_ = true;

protected:
    // CachingWebUIProvider: build contexts once, they're cached
    void buildContexts(std::vector<WebUIContext>& contexts) override {
        for (const auto& ctx : pendingContexts_) {
            contexts.push_back(ctx);
        }
    }

public:
    MockWebUIProvider(const String& n, const String& v) : name_(n), version_(v) {}

    void addContext(const WebUIContext& ctx) {
        pendingContexts_.push_back(ctx);
        // Invalidate cache if already built
        invalidateContextCache();
    }

    String getWebUIName() const override { return name_; }
    String getWebUIVersion() const override { return version_; }

    String handleWebUIRequest(const String& contextId, const String& endpoint,
                              const String& method, const std::map<String, String>& params) override {
        return "{\"success\":true}";
    }

    String getWebUIData(const String& contextId) override {
        return "{}";
    }

    bool isWebUIEnabled() override { return enabled_; }
    void setEnabled(bool e) { enabled_ = e; }
};

/**
 * @brief Simulates Standard example with many providers (16 contexts)
 * 
 * This reproduces the OOM scenario on ESP8266 where many providers
 * create many contexts that all need to be serialized together.
 */
class MultiContextProvider : public CachingWebUIProvider {
protected:
    void buildContexts(std::vector<WebUIContext>& ctxs) override {
        // Simulate WifiWebUI (5 contexts)
        ctxs.push_back(WebUIContext::statusBadge("wifi_status", "WiFi", "dc-wifi").withRealTime(2000));
        ctxs.push_back(WebUIContext::statusBadge("ap_status", "AP", "dc-ap").withRealTime(2000));
        ctxs.push_back(WebUIContext{"wifi_component", "WiFi", "dc-wifi", WebUILocation::ComponentDetail}
            .withField(WebUIField("connected", "Connected", WebUIFieldType::Display, "No", "", true))
            .withField(WebUIField("ssid_now", "SSID", WebUIFieldType::Display, "", "", true))
            .withField(WebUIField("ip", "IP", WebUIFieldType::Display, "0.0.0.0", "", true))
            .withRealTime(2000));
        ctxs.push_back(WebUIContext::settings("wifi_sta_settings", "WiFi Network")
            .withField(WebUIField("ssid", "Network SSID", WebUIFieldType::Text, ""))
            .withField(WebUIField("sta_password", "Password", WebUIFieldType::Password, ""))
            .withField(WebUIField("scan_networks", "Scan Networks", WebUIFieldType::Button, ""))
            .withField(WebUIField("networks", "Available Networks", WebUIFieldType::Display, ""))
            .withField(WebUIField("wifi_enabled", "Enable WiFi", WebUIFieldType::Boolean, "false"))
            .withRealTime(2000));
        ctxs.push_back(WebUIContext::settings("wifi_ap_settings", "Access Point (AP)")
            .withField(WebUIField("ap_ssid", "AP SSID", WebUIFieldType::Text, "DomoticsCore-AP"))
            .withField(WebUIField("ap_enabled", "Enable AP", WebUIFieldType::Boolean, "true"))
            .withRealTime(2000));
        
        // Simulate NTPWebUI (4 contexts)
        ctxs.push_back(WebUIContext::headerInfo("ntp_time", "Time", "dc-clock")
            .withField(WebUIField("time", "Time", WebUIFieldType::Display, "--:--:--", "", true))
            .withRealTime(1000));
        ctxs.push_back(WebUIContext::dashboard("ntp_dashboard", "Current Time", "dc-clock")
            .withField(WebUIField("time", "Time", WebUIFieldType::Display, "--:--:--", "", true))
            .withField(WebUIField("date", "Date", WebUIFieldType::Display, "----/--/--", "", true))
            .withField(WebUIField("timezone", "Timezone", WebUIFieldType::Display, "UTC", "", true))
            .withRealTime(1000));
        ctxs.push_back(WebUIContext::settings("ntp_settings", "NTP Configuration")
            .withField(WebUIField("enabled", "Enable NTP Sync", WebUIFieldType::Boolean, "true"))
            .withField(WebUIField("servers", "NTP Servers", WebUIFieldType::Text, "pool.ntp.org"))
            .withField(WebUIField("sync_interval", "Sync Interval (seconds)", WebUIFieldType::Number, "3600")));
        
        // Simulate SystemInfoWebUI (3 contexts)
        ctxs.push_back(WebUIContext::dashboard("system_info", "Device Information")
            .withField(WebUIField("manufacturer", "Manufacturer", WebUIFieldType::Display, "", "", true))
            .withField(WebUIField("firmware", "Firmware", WebUIFieldType::Display, "", "", true))
            .withField(WebUIField("chip", "Chip", WebUIFieldType::Display, "", "", true))
            .withField(WebUIField("revision", "Revision", WebUIFieldType::Display, "", "", true))
            .withField(WebUIField("cpu_freq", "CPU Freq", WebUIFieldType::Display, "", "", true))
            .withField(WebUIField("total_heap", "Total Heap", WebUIFieldType::Display, "", "", true)));
        ctxs.push_back(WebUIContext::dashboard("system_metrics", "System Metrics")
            .withField(WebUIField("cpu_load", "CPU Load", WebUIFieldType::Chart, "", "%"))
            .withField(WebUIField("heap_usage", "Memory Usage", WebUIFieldType::Chart, "", "%"))
            .withRealTime(2000));
        ctxs.push_back(WebUIContext::settings("system_settings", "Device Settings")
            .withField(WebUIField("device_name", "Device Name", WebUIFieldType::Text, "")));
        
        // Simulate RemoteConsoleWebUI (2 contexts)
        ctxs.push_back(WebUIContext{"console_component", "Remote Console", "dc-plug", WebUILocation::ComponentDetail}
            .withField(WebUIField("status", "Status", WebUIFieldType::Display, "Active", "", true))
            .withField(WebUIField("port", "Port", WebUIFieldType::Display, "23 (Telnet)", "", true)));
        ctxs.push_back(WebUIContext::settings("console_settings", "Remote Console")
            .withField(WebUIField("port", "Telnet Port", WebUIFieldType::Display, "23"))
            .withField(WebUIField("protocol", "Protocol", WebUIFieldType::Display, "Telnet")));
        
        // Simulate WebUI builtin (2 contexts)
        ctxs.push_back(WebUIContext::headerInfo("webui_uptime", "Uptime", "dc-clock")
            .withField(WebUIField("uptime", "Uptime", WebUIFieldType::Display, "0s", "", true))
            .withRealTime(5000));
        ctxs.push_back(WebUIContext::settings("webui_settings", "WebUI Settings")
            .withField(WebUIField("theme", "Theme", WebUIFieldType::Select, "auto"))
            .withField(WebUIField("primary_color", "Primary Color", WebUIFieldType::Color, "#007acc")));
    }
public:
    String getWebUIName() const override { return "MultiTest"; }
    String getWebUIVersion() const override { return "1.0.0"; }
    String getWebUIData(const String&) override { return "{}"; }
    String handleWebUIRequest(const String&, const String&, const String&, const std::map<String, String>&) override { return "{}"; }
    bool hasDataChanged(const String&) override { return false; }
};
