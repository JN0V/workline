








































































































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






     * used to read six parameters by their own names, so no read could ever hit:
     * HA settings could not be saved at all, while `setConfig`, the persistence
     * callback and `publishDiscovery()` ran on every request and the answer was
     * `success`. That is BUG-31.
     *
     * A refused field answers with the reason; the page draws it under the
     * field the value was typed into.



































































































