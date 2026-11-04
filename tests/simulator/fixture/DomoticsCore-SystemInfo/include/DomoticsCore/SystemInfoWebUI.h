// The SystemInfo component's web page.
#pragma once
#include <string>

namespace DomoticsCore {

class SystemInfoWebUI {
public:
    std::string getWebUIVersion() const { return "1.1.0"; }
};

}  // namespace DomoticsCore
