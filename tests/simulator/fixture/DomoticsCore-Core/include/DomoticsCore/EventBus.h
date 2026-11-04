// The event bus: subscribe and unsubscribe by id.
#pragma once
#include <cstdint>
#include <functional>
#include <map>
#include <string>

namespace DomoticsCore {

class EventBus {
public:
    using Handler = std::function<void(const std::string&)>;

    uint32_t subscribe(const std::string& topic, Handler h) {
        uint32_t id = nextId++;
        subs_[id] = {topic, std::move(h)};
        return id;
    }

    void unsubscribe(uint32_t id) { subs_.erase(id); }

    void publish(const std::string& topic, const std::string& payload) {
        for (auto& s : subs_)
            if (s.second.first == topic) s.second.second(payload);
    }

private:
    uint32_t nextId = 1;
    std::map<uint32_t, std::pair<std::string, Handler>> subs_;
};

}  // namespace DomoticsCore
