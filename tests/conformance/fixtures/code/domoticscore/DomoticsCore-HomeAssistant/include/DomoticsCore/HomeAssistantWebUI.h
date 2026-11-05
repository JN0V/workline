#pragma once

#include "DomoticsCore/HomeAssistant.h"
// IWebUIProvider.h carries everything this provider uses — CachingWebUIProvider,
// WebUIContext, WebUIField. WebUI.h was included too and pulls
// <ESPAsyncWebServer.h> (WebUI.h:11), which exists on no host toolchain, so the
// provider could not be compiled by any native test until that line went.
#include "DomoticsCore/IWebUIProvider.h"
#include <ArduinoJson.h>
#include <cstdio>
#include <cstring>

namespace DomoticsCore {
namespace Components {
namespace WebUI {

/**
 * @brief WebUI provider for Home Assistant component
 * 
 * Provides web interface for HA configuration, entity management, and statistics.
 * 
 * UI Contexts:
 * - ha_status: Header badge showing connection status and entity count
 * - ha_dashboard: Dashboard card with entity list
 * - ha_settings: Settings card for configuration
 * - ha_detail: Component detail with statistics
 */
class HomeAssistantWebUI : public CachingWebUIProvider {
public:
    /**
     * @brief Construct WebUI provider
     * @param ha Pointer to HomeAssistant component (non-owning)
     */
    explicit HomeAssistantWebUI(HomeAssistant::HomeAssistantComponent* ha) : ha(ha) {}

    /**
     * @brief Set callback for HomeAssistant configuration persistence (optional)
     */
    void setConfigSaveCallback(std::function<void(const HomeAssistant::HAConfig&)> callback) {
        onConfigSaved = callback;
    }

    // ========== IWebUIProvider Interface ==========

    String getWebUIName() const override {
        return ha ? ha->getMetadata().name : String("HomeAssistant");
    }

    String getWebUIVersion() const override {
        return ha ? ha->getMetadata().version : String("1.4.0");
    }

protected:
    void buildContexts(std::vector<WebUIContext>& contexts) override {
        if (!ha) return;

        // Status badge - placeholder values, real values from getWebUIData()
        contexts.push_back(WebUIContext::statusBadge("ha_status", "Home Assistant", "dc-home-assistant")
            .withField(WebUIField("status", "Status", WebUIFieldType::Display, "0 entities", "", true))
            .withRealTime(5000)
            .withAPI("/api/ha/status")
            .withPriority(80));

        // Dashboard card - Entity overview - placeholder values
        WebUIContext dashboard = WebUIContext::dashboard("ha_dashboard", "Home Assistant", "dc-home-assistant");
        dashboard.withField(WebUIField("node_id", "Node ID", WebUIFieldType::Display, "", "", true))
                 .withField(WebUIField("device_name", "Device", WebUIFieldType::Display, "", "", true))
                 .withField(WebUIField("entity_count", "Entities", WebUIFieldType::Display, "0", "", true))
                 .withField(WebUIField("discovery_count", "Discoveries", WebUIFieldType::Display, "0", "", true))
                 .withField(WebUIField("state_updates", "State Updates", WebUIFieldType::Display, "0", "", true))
                 .withField(WebUIField("commands", "Commands", WebUIFieldType::Display, "0", "", true))
                 .withRealTime(5000)
                 .withAPI("/api/ha/dashboard")
                 .withPriority(75);

        contexts.push_back(dashboard);

        // Settings card - placeholder values
        WebUIContext settings = WebUIContext::settings("ha_settings", "Home Assistant Configuration");
        settings.withField(WebUIField("node_id", "Node ID", WebUIFieldType::Text, ""))
                .withField(WebUIField("device_name", "Device Name", WebUIFieldType::Text, ""))
                .withField(WebUIField("manufacturer", "Manufacturer", WebUIFieldType::Text, ""))
                .withField(WebUIField("model", "Model", WebUIFieldType::Text, ""))
                .withField(WebUIField("discovery_prefix", "Discovery Prefix", WebUIFieldType::Text, "homeassistant"))
                .withField(WebUIField("suggested_area", "Suggested Area", WebUIFieldType::Text, ""))
                .withAPI("/api/ha/settings");

        contexts.push_back(settings);

        // Component detail - Full statistics - placeholder values
        contexts.push_back(WebUIContext("ha_detail", "Home Assistant Details", "dc-home-assistant",
                                       WebUILocation::ComponentDetail)
            .withField(WebUIField("entity_count", "Total Entities", WebUIFieldType::Display, "0", "", true))
            .withField(WebUIField("discovery_count", "Discovery Publishes", WebUIFieldType::Display, "0", "", true))
            .withField(WebUIField("discovery_refused", "Configs Refused (too long)", WebUIFieldType::Display, "0", "", true))
            .withField(WebUIField("state_updates", "State Updates Sent", WebUIFieldType::Display, "0", "", true))
            .withField(WebUIField("commands_received", "Commands Received", WebUIFieldType::Display, "0", "", true))
            .withField(WebUIField("availability_topic", "Availability Topic", WebUIFieldType::Display, "", "", true))
            .withField(WebUIField("config_url", "Config URL", WebUIFieldType::Display, "N/A", "", true))
            .withRealTime(5000)
            .withAPI("/api/ha/detail"));
    }

public:

    String getWebUIData(const String& contextId) override {
        if (!ha) return "{}";

        JsonDocument doc;
        const auto& cfg = ha->getConfig();
        const auto& stats = ha->getStatistics();

        if (contextId == "ha_status") {
            String statusText = String(stats.entityCount) + " entities";
            doc["status"] = statusText;

        } else if (contextId == "ha_dashboard") {
            doc["node_id"] = (const char*)cfg.nodeId;
            doc["device_name"] = (const char*)cfg.deviceName;
            doc["entity_count"] = stats.entityCount;
            doc["discovery_count"] = stats.discoveryCount;
            doc["state_updates"] = stats.stateUpdates;
            doc["commands"] = stats.commandsReceived;

        } else if (contextId == "ha_settings") {
            doc["node_id"] = (const char*)cfg.nodeId;
            doc["device_name"] = (const char*)cfg.deviceName;
            doc["manufacturer"] = (const char*)cfg.manufacturer;
            doc["model"] = (const char*)cfg.model;
            doc["discovery_prefix"] = (const char*)cfg.discoveryPrefix;
            doc["suggested_area"] = (const char*)cfg.suggestedArea;

        } else if (contextId == "ha_detail") {
            doc["entity_count"] = stats.entityCount;
            doc["discovery_count"] = stats.discoveryCount;
            doc["discovery_refused"] = stats.discoveryRefused;
            doc["state_updates"] = stats.stateUpdates;
            doc["commands_received"] = stats.commandsReceived;
            doc["availability_topic"] = (const char*)cfg.availabilityTopic;
            doc["config_url"] = cfg.configUrl[0] == '\0' ? "N/A" : (const char*)cfg.configUrl;
        }

        return webUIContextData(doc);
    }

    /**
     * @brief Apply one settings field, following the framework's only convention.
     *
     * The dispatcher (`WebUI.h:865-876`) builds exactly two parameters, `field`
     * and `value`, for both the HTTP route and the WebSocket action. This handler
     * used to read six parameters by their own names, so no read could ever hit:
     * HA settings could not be saved at all, while `setConfig`, the persistence
     * callback and `publishDiscovery()` ran on every request and the answer was
     * `success`. That is BUG-31.
     *
     * A refused field answers with the reason; the page draws it under the
     * field the value was typed into.
     */
    String handleWebUIRequest(const String& contextId, const String& endpoint,
                              const String& method, const std::map<String, String>& params) override {
        if (!ha) return "{\"success\":false,\"error\":\"Component not available\"}";
        if (method != "POST") return "{\"success\":false,\"error\":\"Method not allowed\"}";
        if (contextId != "ha_settings") return "{\"success\":false,\"error\":\"Unknown context\"}";

        auto fieldIt = params.find("field");
        auto valueIt = params.find("value");
        if (fieldIt == params.end() || valueIt == params.end()) return "{\"success\":false,\"error\":\"Invalid request\"}";
        const String& field = fieldIt->second;
        const String& value = valueIt->second;

        using namespace HomeAssistant;
        const HAConfig& cur = ha->getConfig();
        HAConfig newCfg = cur;

        // Where the new value goes, what it may not exceed, and what it is today.
        char* dest = nullptr;
        const char* currentValue = nullptr;
        size_t maxLen = 0;
        // node_id and discovery_prefix are what a generated availability topic is
        // built from; moving either has to move the topic with it.
        bool namesTheAvailabilityTopic = false;

        if (field == "node_id") {
            dest = newCfg.nodeId;           currentValue = cur.nodeId;
            maxLen = HA::MAX_NODE_ID;       namesTheAvailabilityTopic = true;
        } else if (field == "device_name") {
            dest = newCfg.deviceName;       currentValue = cur.deviceName;
            maxLen = HA::MAX_DEVICE_NAME;
        } else if (field == "manufacturer") {
            dest = newCfg.manufacturer;     currentValue = cur.manufacturer;
            maxLen = HA::MAX_MANUFACTURER;
        } else if (field == "model") {
            dest = newCfg.model;            currentValue = cur.model;
            maxLen = HA::MAX_MODEL;
        } else if (field == "discovery_prefix") {
            dest = newCfg.discoveryPrefix;  currentValue = cur.discoveryPrefix;
            maxLen = HA::MAX_DISCOVERY_PREFIX; namesTheAvailabilityTopic = true;
        } else if (field == "suggested_area") {
            dest = newCfg.suggestedArea;    currentValue = cur.suggestedArea;
            maxLen = HA::MAX_SUGGESTED_AREA;
        } else {
            return "{\"success\":false,\"error\":\"Unknown field\"}";   // nothing touched
        }

        // Stored, an empty identity would win at every boot: no prefix means no
        // discovery, no node id an empty topic level.
        if (value.length() == 0 && (field == "node_id" || field == "discovery_prefix" ||
                                    field == "device_name")) {
            return "{\"success\":false,\"error\":\"This field cannot be empty\"}";
        }

        // HA::setField is the only validation these fields have: a truncation with
        // a warning. Comparing after it means an over-long value that truncates to
        // what is already stored counts as unchanged, as it should.
        HA::setField(dest, value.c_str(), maxLen);
        if (strcmp(currentValue, dest) == 0) {
            // Nothing moved: no setConfig, no persistence write, no republish.
            return "{\"success\":true}";
        }

        if (namesTheAvailabilityTopic) {
            // setConfig regenerates the topic only when it is empty
            // (HomeAssistant.h:465-475), so a changed node id would otherwise keep
            // publishing availability on the previous node's topic. Clear it only
            // when it is the generated one — a topic somebody set deliberately is
            // not ours to rewrite (test_ha_component.cpp:126-135 pins that).
            char generatedFromOld[HA::MAX_AVAIL_TOPIC];
            snprintf(generatedFromOld, sizeof(generatedFromOld), "%s/%s/availability",
                     cur.discoveryPrefix, cur.nodeId);
            if (cur.availabilityTopic[0] == '\0' ||
                strcmp(cur.availabilityTopic, generatedFromOld) == 0) {
                newCfg.availabilityTopic[0] = '\0';
            }
        }

        ha->setConfig(newCfg);

        // Invoke persistence callback if set
        if (onConfigSaved) {
            onConfigSaved(ha->getConfig());
        }

        // Republish discovery with new configuration
        ha->publishDiscovery();

        return "{\"success\":true}";
    }

private:
    HomeAssistant::HomeAssistantComponent* ha;
    std::function<void(const HomeAssistant::HAConfig&)> onConfigSaved;  // Callback for persistence
};

} // namespace WebUI
} // namespace Components
} // namespace DomoticsCore
