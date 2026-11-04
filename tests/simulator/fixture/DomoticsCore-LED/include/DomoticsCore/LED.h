// The LED component: named LEDs, each on or off.
#pragma once
#include <string>
#include <vector>

namespace DomoticsCore {

struct LEDInfo {
    std::string name;
    bool enabled = false;
};

class LEDComponent {
public:
    void addLED(const std::string& name) { leds_.push_back({name, false}); }
    std::vector<LEDInfo>& leds() { return leds_; }

private:
    std::vector<LEDInfo> leds_;
};

}  // namespace DomoticsCore
