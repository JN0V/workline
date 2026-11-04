// The NTP component's web page.
#pragma once
#include <string>

namespace DomoticsCore {

class NTPWebUI {
public:
    std::string getWebUIVersion() const { return "1.0.2"; }
};

}  // namespace DomoticsCore
