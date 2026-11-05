#pragma once

#include "DomoticsCore/Storage.h"
#include "DomoticsCore/IWebUIProvider.h"
#include <ArduinoJson.h>

namespace DomoticsCore {
namespace Components {
namespace WebUI {

class StorageWebUI : public CachingWebUIProvider {
private:
    StorageComponent* storage; // non-owning

public:
    explicit StorageWebUI(StorageComponent* comp) : storage(comp) {}

    String getWebUIName() const override { return storage ? storage->metadata.name : String("Storage"); }
    String getWebUIVersion() const override { return storage ? storage->metadata.version : String("1.4.0"); }

protected:
    void buildContexts(std::vector<WebUIContext>& ctxs) override {
        if (!storage) return;

        // Component detail - placeholder values, real values from getWebUIData()
        ctxs.push_back(WebUIContext{
            "storage_component", "Storage", "dc-info", WebUILocation::ComponentDetail
        }
        .withField(WebUIField("namespace", "Namespace", WebUIFieldType::Display, "", "", true))
        .withField(WebUIField("entries", "Used Entries", WebUIFieldType::Display, "0", "", true))
        .withField(WebUIField("free_entries", "Free Entries", WebUIFieldType::Display, "0", "", true))
        .withRealTime(5000));

        // Settings section (read-only basics for now)
        ctxs.push_back(WebUIContext::settings("storage_settings", "Storage Settings")
            .withField(WebUIField("namespace", "Namespace", WebUIFieldType::Display, "", "", true))
        );
    }

public:
    String getWebUIData(const String& contextId) override {
        if (!storage) return "{}";

        JsonDocument doc;
        if (contextId == "storage_component") {
            doc["namespace"] = storage->getNamespace();
            doc["entries"] = storage->getEntryCount();
            doc["free_entries"] = storage->getFreeEntries();
        } else if (contextId == "storage_settings") {
            doc["namespace"] = storage->getNamespace();
        }

        return webUIContextData(doc);
    }

    String handleWebUIRequest(const String& /*contextId*/, const String& /*endpoint*/, const String& method, const std::map<String, String>& /*params*/) override {
        if (method != "POST") return "{\"success\":false,\"error\":\"Method not allowed\"}";
        return "{\"success\":false,\"error\":\"Storage has no editable settings\"}";
    }
};

} // namespace WebUI
} // namespace Components
} // namespace DomoticsCore
