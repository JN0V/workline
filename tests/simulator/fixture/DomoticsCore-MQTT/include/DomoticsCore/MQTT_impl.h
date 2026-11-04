// MQTTComponent's members, inline.
#pragma once

namespace DomoticsCore {

inline void MQTTComponent::connect() {
    state_ = MQTTState::Connected;
    ++stats_.connects;
}

// resetReconnect(): closes the session to reconnect at once.
inline void MQTTComponent::resetReconnect() { state_ = MQTTState::Disconnected; }

// announceConnectionLost(): zeroes the uptime.
inline void MQTTComponent::announceConnectionLost() { stats_.uptime = 0; }

}  // namespace DomoticsCore
