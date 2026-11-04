// The RemoteConsole component's web page.
#pragma once
#include <string>

namespace DomoticsCore {

class RemoteConsoleWebUI {
public:
    std::string getWebUIVersion() const { return "1.0.0"; }
};

}  // namespace DomoticsCore
