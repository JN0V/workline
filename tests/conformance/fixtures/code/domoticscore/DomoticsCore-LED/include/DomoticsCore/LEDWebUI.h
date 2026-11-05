#pragma once

#include <DomoticsCore/Platform_HAL.h>
#include <DomoticsCore/IWebUIProvider.h>
#include <DomoticsCore/LED.h>
#include <ArduinoJson.h>
#include <DomoticsCore/BaseWebUIComponents.h>

namespace DomoticsCore {
namespace Components {

// Simple WebUI provider wrapper for LEDComponent
class LEDWebUI : public CachingWebUIProvider {
private:
    LEDComponent* led; // non-owning
    // Simple mirrored state for UI
    size_t selected = 0;
    bool enabled = false; // start OFF by default
    uint8_t brightness = 128;
    LEDEffect effect = LEDEffect::Solid;
    bool initialApplied = false;

    // Cached LED names for context building
    mutable std::vector<String> cachedNames_;
    mutable bool namesCached_ = false;

    void ensureInitialized() {
        if (!initialApplied && led && led->getLEDCount() > 0) {
            selected = 0;
            if (enabled) {
                led->enableLED(selected, true);
                led->setLED(selected, LEDColor::White(), brightness);
                if (effect != LEDEffect::Solid) {
                    led->setLEDEffect(selected, effect, 1000);
                }
            } else {
                led->setLED(selected, LEDColor::Off(), 0);
                led->enableLED(selected, false);
            }
            initialApplied = true;
        }
    }

    std::vector<String> liveNames() const {
        std::vector<String> names = led->getLEDNames();
        if (names.empty()) {
            size_t count = led->getLEDCount();
            for (size_t i = 0; i < count; ++i) {
                names.push_back(String("LED_") + String(i));
            }
        }
        return names;
    }

    // The schema holds the names it was built with; when the live list has
    // moved on (addLED, rename), drop both caches so it is rebuilt.
    void refreshIfNamesChanged() {
        if (!led || !contextsCached_ || !namesCached_) return;
        if (liveNames() != cachedNames_) {
            namesCached_ = false;
            invalidateContextCache();
        }
    }

    const std::vector<String>& getCachedNames() const {
        if (!namesCached_ && led) {
            cachedNames_ = liveNames();
            namesCached_ = true;
        }
        return cachedNames_;
    }

public:
    explicit LEDWebUI(LEDComponent* comp) : led(comp) {}

    String getWebUIName() const override { return led ? led->metadata.name : String("LED"); }
    String getWebUIVersion() const override { return led ? led->metadata.version : String("1.3.0"); }

protected:
    void buildContexts(std::vector<WebUIContext>& contexts) override {
        if (!led) return;

        // Ensure initial state is applied
        ensureInitialized();

        // Status badge showing ON/OFF quickly at the top - placeholder values
        contexts.push_back(WebUIContext::statusBadge("led_status", "LED", "bulb-twotone")
            .withField(WebUIField("state", "State", WebUIFieldType::Status, "OFF"))
            .withRealTime(1000)
            .withCustomCss(R"(
                .status-indicator[data-context-id='led_status'] .status-icon { color: var(--text-secondary); }
                .status-indicator[data-context-id='led_status'].active .status-icon { color: #ffc107; filter: drop-shadow(0 0 6px rgba(255,193,7,0.6)); }
            )"));

        // Dashboard: selection + primary controls
        // Build LED names list for select options (cached)
        const auto& names = getCachedNames();
        contexts.push_back(WebUIContext::dashboard("led_dashboard", "LED Control")
            .withField(WebUIField("led_select", "LED", WebUIFieldType::Select, names.empty() ? "" : names[0]).choices(names))
            .withField(WebUIField("enabled_toggle", "Enabled", WebUIFieldType::Boolean, "false"))
            .withField(WebUIField("brightness", "Brightness", WebUIFieldType::Slider, "128").range(0, 255))
            .withField(WebUIField("effect", "Effect", WebUIFieldType::Select, "Solid").choices({"Solid","Blink","Fade","Pulse","Rainbow","Breathing"}))
            .withRealTime(1000));
    }

public:
    void forEachContext(std::function<bool(const WebUIContext&)> callback) override {
        refreshIfNamesChanged();
        CachingWebUIProvider::forEachContext(callback);
    }
    size_t getContextCount() override {
        refreshIfNamesChanged();
        return CachingWebUIProvider::getContextCount();
    }
    bool getContextAt(size_t index, WebUIContext& outContext) override {
        refreshIfNamesChanged();
        return CachingWebUIProvider::getContextAt(index, outContext);
    }
    const WebUIContext* getContextAtRef(size_t index) override {
        refreshIfNamesChanged();
        return CachingWebUIProvider::getContextAtRef(index);
    }
    WebUIContext getWebUIContext(const String& contextId) override {
        refreshIfNamesChanged();
        return CachingWebUIProvider::getWebUIContext(contextId);
    }

    String getWebUIData(const String& contextId) override {
        // Without a component there is nothing to report, and the dashboard
        // branch below reads the LED names straight off it.
        if (!led) return "{}";

        JsonDocument doc;
        if (contextId == "led_dashboard") {
            // reflect current LED name as the selected value
            {
                auto names = led->getLEDNames();
                if (!names.empty()) {
                    size_t idx = selected < names.size() ? selected : 0;
                    doc["led_select"] = names[idx];
                } else {
                    doc["led_select"] = "";
                }
            }
            doc["enabled_toggle"] = enabled;
            doc["brightness"] = (int)brightness;
            doc["effect"] = effectToString(effect);
        } else if (contextId == "led_status") {
            doc["state"] = enabled ? "ON" : "OFF";
        }
        return webUIContextData(doc);
    }

    String handleWebUIRequest(const String& contextId, const String& endpoint, const String& method, const std::map<String, String>& params) override {
        if (!led) return "{\"success\":false,\"error\":\"Component not available\"}";
        if (method != "POST") return "{\"success\":false,\"error\":\"Method not allowed\"}";
        if (contextId != "led_dashboard") return "{\"success\":false,\"error\":\"Unknown context\"}";
        auto fieldIt = params.find("field"); auto valueIt = params.find("value");
        if (fieldIt == params.end() || valueIt == params.end()) return "{\"success\":false,\"error\":\"Invalid request\"}";
        const String& field = fieldIt->second;
        const String& value = valueIt->second;

        // Helper to clamp selected index
        auto clampIndex = [this](size_t idx){ size_t n = led->getLEDCount(); return (n == 0) ? (size_t)0 : (idx < n ? idx : n-1); };

        if (field == "led_select") {
            // The value is the LED name; one that matches none is refused
            // rather than leaving the selection where it was.
            auto names = led->getLEDNames();
            for (size_t i = 0; i < names.size(); ++i) {
                if (names[i] == value) {
                    selected = clampIndex(i);
                    return "{\"success\":true}";
                }
            }
            return "{\"success\":false,\"error\":\"Unknown LED\"}";
        }
        if (field == "enabled_toggle") {
            // Same vocabulary as the WiFi provider for true; the matching
            // false words are accepted, anything else is refused.
            bool on;
            if (value == "true" || value == "1" || value == "on") on = true;
            else if (value == "false" || value == "0" || value == "off") on = false;
            else return "{\"success\":false,\"error\":\"Invalid boolean value\"}";
            enabled = on;
            // Ensure selected index is valid before using
            selected = clampIndex(selected);
            led->enableLED(selected, enabled);
            if (enabled) {
                // Apply brightness/color immediately when enabling
                led->setLED(selected, LEDColor::White(), brightness);
                if (effect != LEDEffect::Solid) {
                    led->setLEDEffect(selected, effect, 1000);
                }
            } else {
                // Turn off output when disabling
                led->setLED(selected, LEDColor::Off(), 0);
            }
            return "{\"success\":true}";
        }
        if (field == "brightness") {
            // The field is unsigned, so a malformed value is refused rather
            // than coerced to one; a value that parses and overshoots is
            // clamped, which is what a slider does.
            uint32_t b = 0;
            if (!webUIParseUnsigned(value, b)) return "{\"success\":false,\"error\":\"Invalid brightness\"}";
            if (b > 255) b = 255;
            brightness = (uint8_t)b;
            // Ensure selected index is valid before using
            selected = clampIndex(selected);
            // Keep current color intent: ON -> white at brightness, OFF -> off
            if (enabled) led->setLED(selected, LEDColor::White(), brightness);
            else         led->setLED(selected, LEDColor::Off(), 0);
            return "{\"success\":true}";
        }
        if (field == "effect") {
            LEDEffect parsed;
            if (!parseEffect(value, parsed)) return "{\"success\":false,\"error\":\"Unknown effect\"}";
            effect = parsed;
            // Ensure selected index is valid before using
            selected = clampIndex(selected);
            // Apply with default speed
            if (effect == LEDEffect::Solid) {
                if (enabled) led->setLED(selected, LEDColor::White(), brightness);
                else         led->setLED(selected, LEDColor::Off(), 0);
            } else {
                led->setLEDEffect(selected, effect, 1000);
            }
            return "{\"success\":true}";
        }
        return "{\"success\":false,\"error\":\"Unknown field\"}";
    }

private:
    static String effectToString(LEDEffect e) {
        switch (e) {
            case LEDEffect::Solid: return "Solid";
            case LEDEffect::Blink: return "Blink";
            case LEDEffect::Fade: return "Fade";
            case LEDEffect::Pulse: return "Pulse";
            case LEDEffect::Rainbow: return "Rainbow";
            case LEDEffect::Breathing: return "Breathing";
            default: return "Solid";
        }
    }
    static bool parseEffect(const String& s, LEDEffect& out) {
        if (s == "Solid")          { out = LEDEffect::Solid;     return true; }
        if (s == "Blink")          { out = LEDEffect::Blink;     return true; }
        if (s == "Fade")           { out = LEDEffect::Fade;      return true; }
        if (s == "Pulse")          { out = LEDEffect::Pulse;     return true; }
        if (s == "Rainbow")        { out = LEDEffect::Rainbow;   return true; }
        if (s == "Breathing")      { out = LEDEffect::Breathing; return true; }
        return false;
    }
};

} // namespace Components
} // namespace DomoticsCore
