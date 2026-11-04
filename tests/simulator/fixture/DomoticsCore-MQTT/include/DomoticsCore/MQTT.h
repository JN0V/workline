// The MQTT component: its link to the broker and its statistics.
#pragma once
#include <cstdint>

namespace DomoticsCore {

struct MQTTStatistics {
    uint32_t connects = 0;
    uint32_t uptime = 0;
};

enum class MQTTState { Disconnected, Connected };

class MQTTComponent {
public:
    void connect();
    void resetReconnect();
    void announceConnectionLost();
    const MQTTStatistics& stats() const { return stats_; }
    MQTTState state() const { return state_; }

private:
    MQTTStatistics stats_;
    MQTTState state_ = MQTTState::Disconnected;
};

}  // namespace DomoticsCore

#include "DomoticsCore/MQTT_impl.h"
