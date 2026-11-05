// MQTT Component Inline Implementations
// This file is included at the end of MQTT.h

namespace DomoticsCore {
namespace Components {

// A local static of an inline function is one object across translation units,
// which a C++14 build cannot get from a static data member defined in a header.
inline MQTTComponent*& MQTTComponent::instance() {
    static MQTTComponent* current = nullptr;
    return current;
}

// Constructor
inline MQTTComponent::MQTTComponent(const MQTTConfig& cfg)
    : config(cfg)
    , mqttClient(new HAL::MQTT::MQTTClientImpl(config.useTLS, config.caCert))
    , clientTLS_(config.useTLS)
    , clientCa_(config.caCert)
    , state(MQTTState::Disconnected)
    , reconnectTimer(cfg.reconnectDelay)
    , stateChangeTime(0)
    , lastPublishTime(0)
    , publishCountThisSecond(0)
{
    instance() = this;

    normalizeConfig(String());

    // Initialize metadata
    metadata.name = "MQTT";
    metadata.version = "1.11.0";  // x-release-please-version
    metadata.author = "DomoticsCore";
    metadata.description = "MQTT client with auto-reconnection";
}

// Destructor
inline MQTTComponent::~MQTTComponent() {
    if (isConnected()) {
        disconnect();
    }
    delete mqttClient;
    mqttClient = nullptr;
    instance() = nullptr;
}

// Lifecycle methods
inline ComponentStatus MQTTComponent::begin() {
    DLOG_I(LOG_MQTT, "Initializing");
    
    // CRITICAL: Register EventBus listeners FIRST, even if no config yet
    // This ensures listeners are ready when broker gets configured dynamically
    on<MQTTPublishEvent>(MQTTEvents::EVENT_PUBLISH, [this](const MQTTPublishEvent& ev) {
        publish(ev.topic, ev.payload, ev.qos, ev.retain);
    });

    on<MQTTSubscribeEvent>(MQTTEvents::EVENT_SUBSCRIBE, [this](const MQTTSubscribeEvent& ev) {
        subscribe(ev.topic, ev.qos);
    });
    
    DLOG_D(LOG_MQTT, "EventBus listeners registered (mqtt/publish, mqtt/subscribe)");
    
    // CRITICAL: Set PubSubClient callback BEFORE config check
    // This ensures callback is ready when broker gets configured dynamically
    mqttClient->setCallback(mqttCallback);
    DLOG_D(LOG_MQTT, "PubSubClient callback registered");
    
    // Config is loaded by SystemPersistence via setConfig()
    
    // Early return if no broker configured - MQTT stays inactive until configured
    if (config.broker.isEmpty()) {
        DLOG_I(LOG_MQTT, "No broker configured - MQTT inactive until configured");
        return ComponentStatus::Success;  // Success but inactive
    }

    // BUG-8: Copy broker into persistent buffer for PubSubClient.
    // Must happen before the enabled check so that a later connect() call
    // can find the broker even if the component was disabled at begin() time.
    if (config.broker.length() >= sizeof(brokerBuffer_)) {
        DLOG_W(LOG_MQTT, "Broker address truncated: '%s' exceeds %u chars",
               config.broker.c_str(), (unsigned)(sizeof(brokerBuffer_) - 1));
    }
    strncpy(brokerBuffer_, config.broker.c_str(), sizeof(brokerBuffer_) - 1);
    brokerBuffer_[sizeof(brokerBuffer_) - 1] = '\0';
    mqttClient->setServer(brokerBuffer_, config.port);

    if (!config.enabled) {
        DLOG_I(LOG_MQTT, "Component disabled in configuration");
        return ComponentStatus::Success;  // Success but inactive
    }
    // Keep-alive and buffer size are set at connection
    
    // Auto-connect if enabled (components must work independently)
    // System.h can ALSO trigger via WiFi events for better orchestration
    if (config.autoReconnect) {
        connect();
    }
    
    DLOG_I(LOG_MQTT, "Initialized with broker %s:%d, client ID: %s", 
           config.broker.c_str(), config.port, config.clientId.c_str());
    DLOG_I(LOG_MQTT, "MQTT buffer size: %d bytes", MQTT_MAX_PACKET_SIZE);
    
    return ComponentStatus::Success;
}

// Leaving the connected state without disconnect() having been called: move the
// state and say so, so a subscriber's view of the link matches the component's.
inline void MQTTComponent::announceConnectionLost() {
    state = MQTTState::Disconnected;
    stateChangeTime = HAL::Platform::getMillis();
    DLOG_W(LOG_MQTT, "Connection lost");
    emit(MQTTEvents::EVENT_DISCONNECTED, true);
}

inline void MQTTComponent::loop() {
    if (config.broker.isEmpty()) {
        // Clearing the broker at runtime abandons a live session, and nothing
        // below this return would ever notice it leaving Connected.
        if (state == MQTTState::Connected) announceConnectionLost();
        return;
    }

    if (isConnected()) {
        if (reopenPending_) {
            disconnect();
            rebuildClientIfNeeded();
            connect();
            return;
        }
        // Always process an active connection even if config.enabled was
        // cleared after connection (e.g. by a config reload from flash).
        mqttClient->loop();
        updateStatistics();
        processMessageQueue();
        return;
    }

    // A link the broker or the network drops never reaches disconnect(), so the
    // loss is noticed here, at the transition out of Connected, and announced
    // before a reconnection attempt can announce its own success.
    if (state == MQTTState::Connected) announceConnectionLost();

    rebuildClientIfNeeded();
    if (config.enabled && config.autoReconnect) {
        // Only attempt reconnection when explicitly enabled
        handleReconnection();
    }
}

inline ComponentStatus MQTTComponent::shutdown() {
    DLOG_I(LOG_MQTT, "Shutting down");
    
    if (isConnected()) {
        disconnect();
    }
    
    return ComponentStatus::Success;
}

// Connection management
inline bool MQTTComponent::connect() {
    if (isConnected()) {
        DLOG_W(LOG_MQTT, "Already connected");
        return true;
    }
    
    // Check if WiFi is connected before attempting MQTT connection
    if (!HAL::WiFiHAL::isConnected()) {
        lastError = "WiFi not connected";
        DLOG_D(LOG_MQTT, "Cannot connect to MQTT - WiFi not connected");
        return false;
    }
    
    if (config.broker.isEmpty()) {
        lastError = "No broker configured";
        return false;
    }

    // Without a CA every handshake fails; say why instead of retrying it.
    if (config.useTLS && !config.caCert) {
        lastError = "TLS needs a CA certificate";
        return false;
    }
    
    state = MQTTState::Connecting;
    stateChangeTime = HAL::Platform::getMillis();
    // Cleared before the attempt: a setConfig() while it blocks marks it again.
    reopenPending_ = false;

    bool success = connectInternal();
    
    if (success) {
        lastError = "";  // what failed before this session no longer applies
        state = MQTTState::Connected;
        stateChangeTime = HAL::Platform::getMillis();
        reconnectTimer.setInterval(config.reconnectDelay);  // Reset to initial delay
        stats.connectCount++;
        stats.reconnectCount = 0;  // Reset failure counter on successful connection
        
        DLOG_I(LOG_MQTT, "Connected to %s:%d", config.broker.c_str(), config.port);
        
        // Resubscribe to all topics
        for (const auto& sub : subscriptions) {
            mqttClient->subscribe(sub.topic.c_str(), sub.qos);
        }
        
        // Emit event for decoupled components
        emit(MQTTEvents::EVENT_CONNECTED, true);
    } else {
        state = MQTTState::Error;
        stateChangeTime = HAL::Platform::getMillis();
        lastError = "Connection failed";
        // A wrong CA and a name the certificate does not carry look alike otherwise.
        char tls[96];
        if (config.useTLS && mqttClient->lastTlsError(tls, sizeof(tls)) != 0) {
            lastError = String("TLS: ") + tls;
        }
        DLOG_E(LOG_MQTT, "%s", lastError.c_str());
    }
    
    return success;
}

inline void MQTTComponent::disconnect() {
    if (!isConnected()) return;

    mqttClient->disconnect();
    state = MQTTState::Disconnected;
    stateChangeTime = HAL::Platform::getMillis();

    DLOG_I(LOG_MQTT, "Disconnected from broker");

    // Emit event for decoupled components
    emit(MQTTEvents::EVENT_DISCONNECTED, true);
}

inline void MQTTComponent::resetReconnect() {
    stats.reconnectCount = 0;
    reconnectTimer.setInterval(config.reconnectDelay);
    reconnectTimer.enable();
    reconnectTimer.reset();
    state = MQTTState::Disconnected;
    lastError = "";
    DLOG_I(LOG_MQTT, "Reconnection reset - auto-retry re-enabled");
}

inline String MQTTComponent::getStateString() const {
    switch (state) {
        case MQTTState::Disconnected: return "Disconnected";
        case MQTTState::Connecting: return "Connecting";
        case MQTTState::Connected: return "Connected";
        case MQTTState::Error: return "Error";
        default: return "Unknown";
    }
}

// Publishing
inline bool MQTTComponent::rateLimitAllowsPublish() {
    if (config.publishRateLimit == 0) return true;  // 0 = unlimited

    // Tumbling window: reset the counter once the current second has elapsed.
    unsigned long now = HAL::getMillis();
    if (now - lastPublishTime >= 1000) {
        publishCountThisSecond = 0;
        lastPublishTime = now;
    }
    return publishCountThisSecond < config.publishRateLimit;
}

inline bool MQTTComponent::enqueueMessage(const String& topic, const String& payload,
                                          uint8_t qos, bool retain) {
    // Queue size guard (0 = unlimited)
    if (config.maxQueueSize > 0 && messageQueue.size() >= config.maxQueueSize) {
        DLOG_W(LOG_MQTT, "Message queue full (%u/%u), dropping message for '%s'",
               (unsigned)messageQueue.size(), config.maxQueueSize, topic.c_str());
        stats.publishErrors++;
        return false;
    }
    // Counted in bytes too: a hundred large payloads exhaust an ESP8266's heap long before the count.
    const size_t cost = queueCost(topic, payload);
    if (queuedBytes + cost > HAL::MQTT::kQueueByteBudget) {
        DLOG_W(LOG_MQTT, "Message queue over its %u-byte budget (%u held), dropping message for '%s'",
               (unsigned)HAL::MQTT::kQueueByteBudget, (unsigned)queuedBytes, topic.c_str());
        stats.publishErrors++;
        return false;
    }
    messageQueue.push_back({topic, payload, qos, retain});
    queuedBytes += cost;
    return true;
}

inline bool MQTTComponent::publish(const String& topic, const String& payload, uint8_t qos, bool retain) {
    if (qos > 2) {
        DLOG_W(LOG_MQTT, "Invalid QoS %u for publish, clamping to 2", qos);
        qos = 2;
    }

    // BUG-29: over the rate limit, defer rather than discard. This used to drop
    // the message, which cost a device its last entities: publishDiscovery()
    // sends one config per entity back to back, so anything past the tenth in the
    // same second vanished. Order-dependent, and silent — adding a sensor could
    // remove an unrelated button, and Home Assistant simply never saw it.
    //
    // The queue that makes this work is the one the offline path already uses,
    // and processMessageQueue() already drains it on every connected loop().
    if (!rateLimitAllowsPublish()) {
        DLOG_D(LOG_MQTT, "Publish rate limit reached (%u/s), queueing '%s' for the next window",
               config.publishRateLimit, topic.c_str());
        return enqueueMessage(topic, payload, qos, retain);
    }

    if (!isConnected()) {
        // Queue message for later if offline. Note this deliberately does not
        // advance publishCountThisSecond — nothing has gone out on the wire, and
        // counting here would charge the message twice, once now and once when
        // the queue actually drains it.
        return enqueueMessage(topic, payload, qos, retain);
    }

    // Older messages leave first: a state must not overtake its entity's discovery document.
    if (!messageQueue.empty()) return enqueueMessage(topic, payload, qos, retain);

    if (!packetFits(topic.c_str(), payload.length())) return false;
    // A socket still waiting for the broker's acknowledgements would block loop().
    if (!mqttClient->canWrite(packetLength(topic.c_str(), payload.length()))) {
        return enqueueMessage(topic, payload, qos, retain);
    }
    return writeNow(topic, payload, retain);
}

inline bool MQTTComponent::writeNow(const String& topic, const String& payload, bool retain) {
    DLOG_D(LOG_MQTT, "Publishing to topic '%s' (retain %s), size: %d bytes",
           topic.c_str(), retain ? "true" : "false", payload.length());
    bool success = mqttClient->publish(topic.c_str(), (const uint8_t*)payload.c_str(), payload.length(), retain);

    if (success) {
        stats.publishCount++;
        publishCountThisSecond++;
        DLOG_D(LOG_MQTT, "  ✓ Published successfully");
    } else {
        stats.publishErrors++;
        DLOG_E(LOG_MQTT, "  ✗ Publish failed! Client state: %d, buffer size: %d",
               mqttClient->state(), mqttClient->getBufferSize());
    }
    
    return success;
}

inline bool MQTTComponent::publishNow(const char* topic, const char* payload, size_t len, bool retain) {
    if (!topic || !payload || !mqttClient) return false;
    if (!isConnected() || !rateLimitAllowsPublish()) return false;
    if (!packetFits(topic, len)) return false;
    if (!mqttClient->canWrite(packetLength(topic, len))) return false;
    bool success = mqttClient->publish(topic, reinterpret_cast<const uint8_t*>(payload), len, retain);
    if (success) {
        stats.publishCount++;
        publishCountThisSecond++;
    } else {
        stats.publishErrors++;
    }
    return success;
}

inline bool MQTTComponent::publishJSON(const String& topic, const JsonDocument& doc, uint8_t qos, bool retain) {
    String payload;
    serializeJson(doc, payload);
    return publish(topic, payload, qos, retain);
}

inline bool MQTTComponent::publishBinary(const String& topic, const uint8_t* data, size_t length, uint8_t qos, bool retain) {
    if (!isConnected()) return false;
    if (!mqttClient->canWrite(packetLength(topic.c_str(), length))) return false;

    bool success = mqttClient->publish(topic.c_str(), data, length, retain);

    if (success) {
        stats.publishCount++;
    } else {
        stats.publishErrors++;
    }

    return success;
}

// Subscribing
inline bool MQTTComponent::subscribe(const String& topic, uint8_t qos) {
    if (qos > 2) {
        DLOG_W(LOG_MQTT, "Invalid QoS %u for subscribe, clamping to 2", qos);
        qos = 2;
    }
    // Check if already subscribed
    for (const auto& sub : subscriptions) {
        if (sub.topic == topic) {
            return true;
        }
    }
    
    // Max subscriptions guard (0 = unlimited)
    if (config.maxSubscriptions > 0 && subscriptions.size() >= config.maxSubscriptions) {
        DLOG_W(LOG_MQTT, "Max subscriptions reached (%u/%u), rejecting '%s'",
               (unsigned)subscriptions.size(), config.maxSubscriptions, topic.c_str());
        return false;
    }

    if (!isConnected()) {
        subscriptions.push_back({topic, qos});
        stats.subscriptionCount = subscriptions.size();
        return true;
    }

    bool success = mqttClient->subscribe(topic.c_str(), qos);

    if (success) {
        subscriptions.push_back({topic, qos});
        stats.subscriptionCount = subscriptions.size();
        DLOG_I(LOG_MQTT, "Subscribed to: %s (QoS %d)", topic.c_str(), qos);
    }

    return success;
}

inline bool MQTTComponent::unsubscribe(const String& topic) {
    bool success = mqttClient->unsubscribe(topic.c_str());

    if (success) {
        auto it = subscriptions.begin();
        while (it != subscriptions.end()) {
            if (it->topic == topic) {
                it = subscriptions.erase(it);
                break;
            } else {
                ++it;
            }
        }
        stats.subscriptionCount = subscriptions.size();
        subscriptions.shrink_to_fit();
        DLOG_I(LOG_MQTT, "Unsubscribed from: %s", topic.c_str());
    }

    return success;
}

inline void MQTTComponent::unsubscribeAll() {
    for (const auto& sub : subscriptions) {
        mqttClient->unsubscribe(sub.topic.c_str());
    }
    subscriptions.clear();
    subscriptions.shrink_to_fit();
    stats.subscriptionCount = 0;
}

inline std::vector<String> MQTTComponent::getActiveSubscriptions() const {
    std::vector<String> result;
    for (const auto& sub : subscriptions) {
        result.push_back(sub.topic);
    }
    return result;
}

// Callbacks removed - use EventBus for inter-component communication

// Configuration
inline void MQTTComponent::setConfig(const MQTTConfig& cfg) {
    // Preserve enabled flag if we already have an active connection.
    // A config reload from flash must not silently disable message processing.
    bool preserveEnabled = (state == MQTTState::Connected && !cfg.enabled);
    const MQTTConfig previous = config;
    config = cfg;
    normalizeConfig(previous.clientId);
    // The session carries these from CONNECT on; loop() reopens it rather than
    // this caller, which may run on the web server's task.
    const bool sessionLive = (state == MQTTState::Connected || state == MQTTState::Connecting);
    if (sessionLive && !sameSession(previous, config)) {
        DLOG_I(LOG_MQTT, "Session settings changed: reopening the connection");
        reopenPending_ = true;
    }
    if (preserveEnabled) {
        DLOG_W(LOG_MQTT, "setConfig: preserving enabled=true for active connection");
        config.enabled = true;
    }
    // BUG-8: Copy broker into persistent buffer. PubSubClient stores raw pointer.
    if (!config.broker.isEmpty()) {
        if (config.broker.length() >= sizeof(brokerBuffer_)) {
            DLOG_W(LOG_MQTT, "Broker address truncated: '%s' exceeds %u chars",
                   config.broker.c_str(), (unsigned)(sizeof(brokerBuffer_) - 1));
        }
        strncpy(brokerBuffer_, config.broker.c_str(), sizeof(brokerBuffer_) - 1);
        brokerBuffer_[sizeof(brokerBuffer_) - 1] = '\0';
        mqttClient->setServer(brokerBuffer_, config.port);
    }
    // Config persistence handled externally (SystemPersistence)
}

inline void MQTTComponent::setBroker(const String& broker, uint16_t port) {
    config.broker = broker;
    config.port = port;
    // BUG-8: Copy broker into persistent buffer. PubSubClient stores raw pointer.
    if (broker.length() >= sizeof(brokerBuffer_)) {
        DLOG_W(LOG_MQTT, "Broker address truncated: '%s' exceeds %u chars",
               broker.c_str(), (unsigned)(sizeof(brokerBuffer_) - 1));
    }
    strncpy(brokerBuffer_, broker.c_str(), sizeof(brokerBuffer_) - 1);
    brokerBuffer_[sizeof(brokerBuffer_) - 1] = '\0';
    mqttClient->setServer(brokerBuffer_, port);
}

inline void MQTTComponent::setCredentials(const String& username, const String& password) {
    config.username = username;
    config.password = password;
}

// Note: getName() already defined inline in MQTT.h

// Private methods
inline bool MQTTComponent::connectInternal() {
    bool success = false;
    // BUG-8: brokerBuffer_ is already populated by begin(), setConfig(), or setBroker().
    // Guard for empty broker to prevent connecting to uninitialized address.
    if (brokerBuffer_[0] == '\0') {
        DLOG_E(LOG_MQTT, "Cannot connect: broker address is empty");
        return false;
    }
    mqttClient->setServer(brokerBuffer_, config.port);

    // Taken from the config at each connection: persistence applies it after begin().
    mqttClient->setKeepAlive(config.keepAlive);
    mqttClient->setBufferSize(MQTT_MAX_PACKET_SIZE);
    DLOG_D(LOG_MQTT, "MQTT buffer size set to %d bytes", MQTT_MAX_PACKET_SIZE);
    
    // Yield before blocking connection to prevent watchdog issues
    HAL::Platform::yield();

    if (config.enableLWT) {
        if (!config.username.isEmpty()) {
            success = mqttClient->connect(
                config.clientId.c_str(),
                config.username.c_str(),
                config.password.c_str(),
                config.lwtTopic.c_str(),
                config.lwtQoS,
                config.lwtRetain,
                config.lwtMessage.c_str()
            );
        } else {
            success = mqttClient->connect(
                config.clientId.c_str(),
                nullptr,
                nullptr,
                config.lwtTopic.c_str(),
                config.lwtQoS,
                config.lwtRetain,
                config.lwtMessage.c_str()
            );
        }
    } else {
        if (!config.username.isEmpty()) {
            success = mqttClient->connect(
                config.clientId.c_str(),
                config.username.c_str(),
                config.password.c_str()
            );
        } else {
            success = mqttClient->connect(config.clientId.c_str());
        }
    }

    // Yield after blocking connection to allow watchdog reset
    HAL::Platform::yield();

    return success;
}

inline void MQTTComponent::handleReconnection() {
    if (state == MQTTState::Connecting) return;
    
    if (!reconnectTimer.isReady()) return;
    
    // Exponential backoff for reconnection delay
    unsigned long currentDelay = reconnectTimer.getInterval();
    if (currentDelay < config.maxReconnectDelay) {
        unsigned long newDelay = currentDelay * 2;
        if (newDelay > config.maxReconnectDelay) {
            newDelay = config.maxReconnectDelay;
        }
        reconnectTimer.setInterval(newDelay);
    }
    
    DLOG_I(LOG_MQTT, "Attempting reconnection (delay: %lu ms)", reconnectTimer.getInterval());
    stats.reconnectCount++;
    
    // Reset timer and attempt connection
    reconnectTimer.reset();
    connect();
}

inline void MQTTComponent::processMessageQueue() {
    if (messageQueue.empty()) return;

    bool erased = false;
    auto it = messageQueue.begin();
    while (it != messageQueue.end() && isConnected()) {
        // Over the rate limit, stop; the next loop() runs in a fresh window.
        if (!rateLimitAllowsPublish()) break;

        // BUG-39: a packet over the client buffer fails on every loop() and
        // would hold everything behind it forever. Drop it, counted and named.
        if (!packetFits(it->topic.c_str(), it->payload.length())) {
            queuedBytes -= queueCost(it->topic, it->payload);
            it = messageQueue.erase(it);
            erased = true;
            continue;
        }
        if (!mqttClient->canWrite(packetLength(it->topic.c_str(), it->payload.length()))) break;
        if (writeNow(it->topic, it->payload, it->retain)) {
            queuedBytes -= queueCost(it->topic, it->payload);
            it = messageQueue.erase(it);
            erased = true;
        } else {
            break;
        }
    }
    if (erased) messageQueue.shrink_to_fit();
}

inline size_t MQTTComponent::packetLength(const char* topic, size_t payloadLen) {
    return strlen(topic) + payloadLen + 7;
}

inline bool MQTTComponent::packetFits(const char* topic, size_t payloadLen) {
    const size_t packet = packetLength(topic, payloadLen);
    if (packet <= mqttClient->getBufferSize()) return true;
    DLOG_W(LOG_MQTT, "'%s': %u-byte packet exceeds the %u-byte client buffer, dropped", topic,
           (unsigned)packet, (unsigned)mqttClient->getBufferSize());
    stats.publishErrors++;
    return false;
}

inline void MQTTComponent::handleIncomingMessage(char* topic, byte* payload, unsigned int length) {
    stats.receiveCount++;

    // Copy data into fixed-size event buffers for safe EventBus transmission
    MQTTMessageEvent ev{};

    // Copy topic (ensure null-termination)
    strncpy(ev.topic, topic, MQTT_EVENT_TOPIC_SIZE - 1);
    ev.topic[MQTT_EVENT_TOPIC_SIZE - 1] = '\0';

    // Copy payload (ensure null-termination)
    size_t copyLen = (length < MQTT_EVENT_PAYLOAD_SIZE - 1) ? length : (MQTT_EVENT_PAYLOAD_SIZE - 1);
    memcpy(ev.payload, payload, copyLen);
    ev.payload[copyLen] = '\0';

    emit(MQTTEvents::EVENT_MESSAGE, ev);
}

inline void MQTTComponent::updateStatistics() {
    if (isConnected()) {
        stats.uptime = (HAL::Platform::getMillis() - stateChangeTime) / 1000;
    }
}

// TLS and its CA are chosen when the client is built, so a change needs a new one.
// Called from loop() only, with no session open: connect() may run on another task.
inline void MQTTComponent::rebuildClientIfNeeded() {
    if ((clientTLS_ == config.useTLS && clientCa_ == config.caCert) || isConnected()) return;
    delete mqttClient;
    mqttClient = new HAL::MQTT::MQTTClientImpl(config.useTLS, config.caCert);
    mqttClient->setCallback(mqttCallback);
    clientTLS_ = config.useTLS;
    clientCa_ = config.caCert;
}

// An empty client id or will topic means the generated one; a will topic that
// was the previous id's default follows a new id.
inline void MQTTComponent::normalizeConfig(const String& previousClientId) {
    if (config.clientId.isEmpty()) {
        config.clientId = generateClientId();
    }
    const bool wasDefaultWill = !previousClientId.isEmpty() &&
                                config.lwtTopic == previousClientId + "/status";
    if (config.enableLWT && (config.lwtTopic.isEmpty() || wasDefaultWill)) {
        config.lwtTopic = config.clientId + "/status";
    }
    if (config.lwtQoS > 2) {
        DLOG_W(LOG_MQTT, "Invalid lwtQoS %u, clamping to 2", config.lwtQoS);
        config.lwtQoS = 2;
    }
}

inline bool MQTTComponent::sameSession(const MQTTConfig& a, const MQTTConfig& b) {
    return a.broker == b.broker && a.port == b.port && a.useTLS == b.useTLS &&
           a.caCert == b.caCert &&
           a.keepAlive == b.keepAlive && a.username == b.username &&
           a.password == b.password && a.clientId == b.clientId &&
           a.enableLWT == b.enableLWT && a.lwtTopic == b.lwtTopic &&
           a.lwtMessage == b.lwtMessage && a.lwtQoS == b.lwtQoS &&
           a.lwtRetain == b.lwtRetain;
}

inline String MQTTComponent::generateClientId() {
    uint64_t chipId = HAL::getChipId();
    char clientId[32];
    snprintf(clientId, sizeof(clientId), "%s-%04x%08x", 
             HAL::getPlatformName(), (uint16_t)(chipId >> 32), (uint32_t)chipId);
    return String(clientId);
}

// Static callback
inline void MQTTComponent::mqttCallback(char* topic, byte* payload, unsigned int length) {
    if (instance()) {
        instance()->handleIncomingMessage(topic, payload, length);
    }
}

// MQTT 3.1.1 filter match, one segment at a time over the raw strings: no copy.
// "a/#" also matches "a", a filter opening on a wildcard never matches a '$' topic,
// and a malformed filter ('#' not last, a wildcard sharing its level) matches nothing.
inline bool MQTTComponent::topicMatches(const String& filter, const String& topic) {
    const char* f = filter.c_str();
    const char* t = topic.c_str();
    if (!f || !t || !*f || !*t) return false;
    if (*t == '$' && (*f == '+' || *f == '#')) return false;

    while (true) {
        if (*f == '#') return f[1] == '\0';
        if (*f == '+') {
            if (f[1] && f[1] != '/') return false;
            while (*t && *t != '/') t++;
            f++;
        } else {
            while (*f && *f != '/' && *f == *t) { f++; t++; }
            if ((*f && *f != '/') || (*t && *t != '/')) return false;
        }
        // Both now sit on a separator or at their end.
        if (*f == '\0') return *t == '\0';
        if (*t == '\0') return f[1] == '#' && f[2] == '\0';
        f++;
        t++;
    }
}

} // namespace Components
} // namespace DomoticsCore
