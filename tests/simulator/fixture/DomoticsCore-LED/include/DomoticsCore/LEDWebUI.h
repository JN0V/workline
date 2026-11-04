// The LED component's web page: a select of the LEDs and a toggle.
#pragma once
#include <string>
#include <vector>
#include "DomoticsCore/LED.h"

namespace DomoticsCore {

class LEDWebUI {
public:
    explicit LEDWebUI(LEDComponent* led) : led_(led) {}

    std::string getWebUIVersion() const { return "1.3.0"; }

    // The names, cached at the first call.
    const std::vector<std::string>& getCachedNames() {
        if (names_.empty())
            for (auto& l : led_->leds()) names_.push_back(l.name);
        return names_;
    }

    // The led_select handler: the LED picked by its index.
    bool handleSelect(int index) {
        if (index < 0 || index >= static_cast<int>(led_->leds().size())) return false;
        selected_ = index;
        return true;
    }

    // The enabled_toggle handler: "true" turns it on, anything else off.
    void handleToggle(const std::string& value) { led_->leds()[selected_].enabled = (value == "true"); }

private:
    LEDComponent* led_;
    std::vector<std::string> names_;
    int selected_ = 0;
};

}  // namespace DomoticsCore
