// The HomeAssistant component's web page.
#pragma once
#include <string>

namespace DomoticsCore {

class HomeAssistantWebUI {
public:
    std::string getWebUIVersion() const { return "1.4.0"; }
};

}  // namespace DomoticsCore
