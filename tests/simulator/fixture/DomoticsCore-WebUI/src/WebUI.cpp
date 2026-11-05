#include "DomoticsCore/WebUI.h"

namespace DomoticsCore {

char WebUIComponent::wsBuffer_[WEBUI_WS_BUFFER_SIZE];
std::mutex WebUIComponent::bufferLock_;

}  // namespace DomoticsCore
