// The web UI: one buffer for live updates, shared by every page.
#pragma once
#include <cstring>
#include <mutex>
#include <string>

#ifndef WEBUI_WS_BUFFER_SIZE
#define WEBUI_WS_BUFFER_SIZE 1024
#endif

namespace DomoticsCore {

class WebUIComponent {
public:
    // The /api/ui/updates poll handler.
    std::string pollUpdates() {
        std::lock_guard<std::mutex> lock(bufferLock_);
        return std::string(wsBuffer_);
    }
    // The live-update broadcast.
    void broadcast(const std::string& update) {
        std::lock_guard<std::mutex> lock(bufferLock_);
        std::strncpy(wsBuffer_, update.c_str(), WEBUI_WS_BUFFER_SIZE - 1);
        wsBuffer_[WEBUI_WS_BUFFER_SIZE - 1] = '\0';
    }

private:
    static char wsBuffer_[WEBUI_WS_BUFFER_SIZE];
    static std::mutex bufferLock_;
};

}  // namespace DomoticsCore
