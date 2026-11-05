#pragma once

#include "DomoticsCore/SystemInfo.h"
#include "DomoticsCore/IWebUIProvider.h"
#include "DomoticsCore/BaseWebUIComponents.h"
#include "DomoticsCore/Platform_HAL.h"  // For getMillis()
#include "DomoticsCore/MemoryManager.h" // For memory profile
#include <ArduinoJson.h>

namespace DomoticsCore {
namespace Components {
namespace WebUI {

// Composition-based WebUI provider for SystemInfoComponent
class SystemInfoWebUI : public CachingWebUIProvider {
private:
    SystemInfoComponent* sys; // non-owning
    std::function<void(const String&)> onDeviceNameChanged; // callback for device name persistence

    // State tracking using LazyState for change detection
    struct SystemInfoState {
        String deviceName;
        String manufacturer;
        String firmwareVersion;
        bool operator==(const SystemInfoState& other) const {
            return deviceName == other.deviceName &&
                   manufacturer == other.manufacturer &&
                   firmwareVersion == other.firmwareVersion;
        }
        bool operator!=(const SystemInfoState& other) const { return !(*this == other); }
    };
    LazyState<SystemInfoState> systemInfoState;

    // The diagnostic flags as the schema was built with them: the data follows the
    // schema, so a flag changed later waits for invalidateContextCache().
    bool showDetailed_ = true;
    bool showMemory_ = true;

protected:
    // CachingWebUIProvider: build contexts once, they're cached
    void buildContexts(std::vector<WebUIContext>& contexts) override {
        if (!sys) return;

        // A field left out of the schema must also be left out of the data.
        showDetailed_ = sys->getConfig().enableDetailedInfo;
        showMemory_ = sys->getConfig().enableMemoryInfo;

        // Dashboard: Static hardware info - placeholder values, real values from getWebUIData()
        WebUIContext info = WebUIContext::dashboard("system_info", "Device Information");
        info.withField(WebUIField("manufacturer", "Manufacturer", WebUIFieldType::Display, "", "", true))
            .withField(WebUIField("firmware", "Firmware", WebUIFieldType::Display, "", "", true));
        if (showDetailed_) {
            info.withField(WebUIField("chip", "Chip", WebUIFieldType::Display, "", "", true))
                .withField(WebUIField("revision", "Revision", WebUIFieldType::Display, "", "", true))
                .withField(WebUIField("cpu_freq", "CPU Freq", WebUIFieldType::Display, "", "", true));
        }
        if (showMemory_) {
            info.withField(WebUIField("total_heap", "Total Heap", WebUIFieldType::Display, "", "", true))
                .withField(WebUIField("mem_profile", "Mem Profile", WebUIFieldType::Display, "", "", true));
        }
        contexts.push_back(info);

        // Dashboard: Real-time metrics with charts
        WebUIContext live = WebUIContext::dashboard("system_metrics", "System Metrics");
        live.withField(WebUIField("cpu_load", "CPU Load", WebUIFieldType::Chart, "", "%"));
        if (showMemory_) {
            live.withField(WebUIField("heap_usage", "Memory Usage", WebUIFieldType::Chart, "", "%"));
        }
        contexts.push_back(live.withRealTime(2000));

        // Settings: Device Name only
        contexts.push_back(WebUIContext::settings("system_settings", "Device Settings")
            .withField(WebUIField("device_name", "Device Name", WebUIFieldType::Text, "")));
    }

public:
    explicit SystemInfoWebUI(SystemInfoComponent* component)
        : sys(component) {}

    // Set callback for device name persistence (optional)
    void setDeviceNameCallback(std::function<void(const String&)> callback) {
        onDeviceNameChanged = callback;
    }

    // IWebUIProvider
    String getWebUIName() const override { return sys ? sys->metadata.name : String("System Info"); }
    String getWebUIVersion() const override { return sys ? sys->metadata.version : String("1.2.1"); }

    String getWebUIData(const String& contextId) override {
        if (!sys) return "{}";
        ensureContextsCached();
        const auto& metrics = sys->getMetrics();
        const auto& cfg = sys->getConfig();

        if (contextId == "system_info") {
            JsonDocument doc;
            doc["manufacturer"] = cfg.manufacturer;
            doc["firmware"] = cfg.firmwareVersion;
            if (showDetailed_) {
                doc["chip"] = metrics.chipModel;
                doc["revision"] = metrics.chipRevision;
                doc["cpu_freq"] = String((uint32_t)metrics.cpuFreq) + " MHz";
            }
            if (showMemory_) {
                doc["total_heap"] = String(metrics.totalHeap / 1024) + " KB";
                doc["mem_profile"] = MemoryManager::instance().getProfileName();
            }
            String json; serializeJson(doc, json); return json;

        } else if (contextId == "system_metrics") {
            JsonDocument doc;
            doc["cpu_load"] = metrics.cpuLoad;
            if (showMemory_) {
                float heapPercent = metrics.totalHeap > 0 ?
                    ((float)(metrics.totalHeap - metrics.freeHeap) / metrics.totalHeap) * 100.0f : 0.0f;
                doc["heap_usage"] = heapPercent;
            }
            String json; serializeJson(doc, json); return json;

        } else if (contextId == "system_settings") {
            JsonDocument doc;
            doc["device_name"] = cfg.deviceName;
            String json; serializeJson(doc, json); return json;
        }

        return "{}";
    }

    String handleWebUIRequest(const String& contextId, const String& /*endpoint*/, const String& method, const std::map<String, String>& params) override {
        // Guard the component, as getWebUIData and hasDataChanged already do:
        // without it a POST dereferences a null sys below and crashes.
        if (!sys) return "{\"success\":false,\"error\":\"Component not available\"}";

        if (contextId != "system_settings") return "{\"success\":false,\"error\":\"Unknown context\"}";
        if (method != "POST") return "{\"success\":false,\"error\":\"Method not allowed\"}";

        auto fieldIt = params.find("field");
        auto valueIt = params.find("value");
        if (fieldIt == params.end() || valueIt == params.end()) return "{\"success\":false,\"error\":\"Invalid request\"}";
        {
            {
                const String& field = fieldIt->second;
                const String& value = valueIt->second;

                if (field == "device_name") {
                    if (value.length() == 0) {
                        return "{\"success\":false,\"error\":\"Device name cannot be empty\"}";
                    }
                    // Cap at 31, the char[31] the browser header and WebUIConfig
                    // already truncate to (WebUIConfig.h:17). SystemConfig keeps the
                    // untruncated String, so this makes two ends agree, not all. The
                    // cap is byte-based, matching WebUIConfig: a UTF-8 name whose 31st
                    // byte is mid-sequence loses that code point, which the escaper
                    // renders as U+FFFD — cosmetic, not corruption, and consistent.
                    String capped = value.length() > 31 ? value.substring(0, 31) : value;

                    // Use Get → Override → Set pattern
                    SystemInfoConfig cfg = sys->getConfig();
                    cfg.deviceName = capped;
                    sys->setConfig(cfg);

                    // Invoke persistence callback
                    if (onDeviceNameChanged) {
                        onDeviceNameChanged(capped);
                    }

                    // Force state reset to push update immediately
                    systemInfoState.reset();

                    return "{\"success\":true}";
                }
            }
        }
        return "{\"success\":false,\"error\":\"Unknown field\"}";
    }

    bool hasDataChanged(const String& contextId) override {
        if (!sys) return false;

        // Static hardware info never changes
        if (contextId == "system_info") {
            return false;
        }

        // Settings change detection
        if (contextId == "system_settings") {
            const auto& cfg = sys->getConfig();
            SystemInfoState current{
                cfg.deviceName,
                cfg.manufacturer,
                cfg.firmwareVersion
            };
            return systemInfoState.hasChanged(current);
        }

        // Metrics always update (real-time data)
        if (contextId == "system_metrics") {
            return true;
        }

        return false;
    }
};

} // namespace WebUI
} // namespace Components
} // namespace DomoticsCore
