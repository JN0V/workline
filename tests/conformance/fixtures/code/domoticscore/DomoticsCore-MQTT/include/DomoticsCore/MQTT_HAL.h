#pragma once

/**
 * @file MQTT_HAL.h
 * @brief Hardware Abstraction Layer for MQTT client
 *
 * Provides platform-agnostic MQTT client interface that wraps
 * Arduino-specific PubSubClient library. Enables native testing
 * and respects Constitution Principle IX (HAL).
 *
 * Platform-specific implementations:
 * - MQTT_ESP32.h: ESP32 with PubSubClient
 * - MQTT_ESP8266.h: ESP8266 with PubSubClient
 * - MQTT_Stub.h: Mock for native tests
 */

#include <DomoticsCore/Platform_HAL.h>

namespace DomoticsCore {
namespace HAL {
namespace MQTT {

/**
 * @brief Platform-agnostic MQTT client interface
 *
 * Abstracts PubSubClient to enable testing and multi-platform support
 */
class MQTTClient {
public:
    virtual ~MQTTClient() = default;

    /**
     * @brief Connect to MQTT broker with full options
     * @param id Client ID
     * @param user Username (nullptr if not used)
     * @param pass Password (nullptr if not used)
     * @param willTopic Last Will Testament topic (nullptr if not used)
     * @param willQoS LWT QoS (0, 1, or 2)
     * @param willRetain LWT retain flag
     * @param willMessage LWT message (nullptr if not used)
     * @return true if connection successful
     */
    virtual bool connect(const char* id,
                        const char* user = nullptr,
                        const char* pass = nullptr,
                        const char* willTopic = nullptr,
                        uint8_t willQoS = 0,
                        bool willRetain = false,
                        const char* willMessage = nullptr) = 0;

    /**
     * @brief Disconnect from broker
     */
    virtual void disconnect() = 0;

    /**
     * @brief Process MQTT messages (call in loop)
     * @return true if connected
     */
    virtual bool loop() = 0;

    /**
     * @brief Publish message to topic
     * @param topic Topic name
     * @param payload Message payload
     * @param length Payload length
     * @param retained Retain flag
     * @return true if publish successful
     */
    virtual bool publish(const char* topic,
                        const uint8_t* payload,
                        unsigned int length,
                        bool retained = false) = 0;

    /**
     * @brief Subscribe to topic
     * @param topic Topic filter
     * @param qos QoS level (0, 1, or 2)
     * @return true if subscribe successful
     */
    virtual bool subscribe(const char* topic, uint8_t qos = 0) = 0;

    /**
     * @brief Unsubscribe from topic
     * @param topic Topic filter
     * @return true if unsubscribe successful
     */
    virtual bool unsubscribe(const char* topic) = 0;

    /**
     * @brief Configure broker address
     * @param domain Broker hostname or IP
     * @param port Broker port
     */
    virtual void setServer(const char* domain, uint16_t port) = 0;

    /**
     * @brief Set message received callback
     * @param callback Function called when message arrives
     */
    virtual void setCallback(void (*callback)(char*, uint8_t*, unsigned int)) = 0;

    /**
     * @brief Set keep-alive interval
     * @param keepAlive Seconds between keep-alive pings
     */
    virtual void setKeepAlive(uint16_t keepAlive) = 0;

    /**
     * @brief Set internal buffer size
     * @param size Buffer size in bytes
     * @return true if buffer size set successfully
     */
    virtual bool setBufferSize(uint16_t size) = 0;

    /**
     * @brief Get current buffer size
     * @return Buffer size in bytes
     */
    virtual uint16_t getBufferSize() = 0;

    /**
     * @brief Get connection state
     * @return State code (PubSubClient state codes)
     */
    virtual int state() = 0;

    /**
     * @brief Check if connected to broker
     * @return true if connected
     */
    virtual bool connected() = 0;

    /**
     * @brief Get number of loop() calls (debug/testing)
     * @return Call count, or 0 if not tracked by this implementation
     */
    virtual uint32_t getLoopCallCount() const { return 0; }

    /**
     * @brief True when a packet of this length can be handed to the network
     * stack without waiting for the broker to acknowledge earlier data.
     */
    virtual bool canWrite(size_t packetLength) { (void)packetLength; return true; }

    /** @brief Why the last TLS handshake failed: a code, 0 for none, and its text in buf. */
    virtual int lastTlsError(char* buf, size_t size) {
        if (buf && size) buf[0] = '\0';
        return 0;
    }
};

} // namespace MQTT
} // namespace HAL
} // namespace DomoticsCore

// Platform-specific implementation routing
#if defined(ARDUINO_ARCH_ESP32)
    #include "MQTT_ESP32.h"
#elif defined(ARDUINO_ARCH_ESP8266) || defined(ESP8266)
    #include "MQTT_ESP8266.h"
#else
    #include "MQTT_Stub.h"
#endif
