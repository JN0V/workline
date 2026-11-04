// The Wifi component's web page.
#pragma once
#include <string>

namespace DomoticsCore {

class WifiWebUI {
public:
    std::string getWebUIVersion() const { return "1.2.1"; }
};

}  // namespace DomoticsCore
