#pragma once

/**
 * @file MQTTEvents.h
 * @brief MQTT component events.
 *
 * Published by MQTTComponent when broker connection state changes.
 * Subscribe to these events to react to MQTT connectivity.
 */

namespace DomoticsCore {
namespace MQTTEvents {

/// Published when connected to MQTT broker
static constexpr const char* EVENT_CONNECTED = "mqtt/connected";

/// Published when disconnected from MQTT broker
static constexpr const char* EVENT_DISCONNECTED = "mqtt/disconnected";

/// Published when a message is received
static constexpr const char* EVENT_MESSAGE = "mqtt/message";

/// Published when a message is successfully published
static constexpr const char* EVENT_PUBLISH = "mqtt/publish";

/// Published when a subscription is confirmed
static constexpr const char* EVENT_SUBSCRIBE = "mqtt/subscribe";

} // namespace MQTTEvents
} // namespace DomoticsCore
