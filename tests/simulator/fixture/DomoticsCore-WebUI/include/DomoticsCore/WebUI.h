// The web UI: one buffer for live updates, shared by every page.
#pragma once
#include <mutex>
#include <string>

namespace DomoticsCore {

class WebUIComponent {
public:
    // The /api/ui/updates poll handler.
    std::string pollUpdates() {
        std::lock_guard<std::mutex> lock(bufferLock_);
        return wsBuffer_;
    }
    // The live-update broadcast.
    void broadcast(const std::string& update) {
        std::lock_guard<std::mutex> lock(bufferLock_);
        wsBuffer_ = update;
    }

private:
    static std::string wsBuffer_;
    static std::mutex bufferLock_;
};

}  // namespace DomoticsCore
