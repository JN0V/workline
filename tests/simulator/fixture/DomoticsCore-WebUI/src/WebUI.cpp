#include "DomoticsCore/WebUI.h"

namespace DomoticsCore {

std::string WebUIComponent::wsBuffer_;
std::mutex WebUIComponent::bufferLock_;

}  // namespace DomoticsCore
