































































































































    // IComponent interface
    ComponentStatus begin() override {
        DLOG_I(LOG_HA, "Initializing Home Assistant integration");
        DLOG_I(LOG_HA, "Node ID: %s", config.nodeId);
        DLOG_I(LOG_HA, "Discovery prefix: %s", config.discoveryPrefix);

        // Drop the link tracker shutdown() left running (resumeConnected is
        // the live link state by now); the real handlers replace it.
        if (__dc_eventBus) __dc_eventBus->unsubscribeOwner(this);

        // Before anything advertises a topic, make it the one the broker corrects.
        reconcileAvailabilityWithWill();
        
        // Subscribe to MQTT events via EventBus
        auto onMqttConnected = [this]() {
            DLOG_I(LOG_HA, "MQTT connected (via EventBus), publishing availability");
            // A new session may carry a will moved since begin(): follow it.
            reconcileAvailabilityWithWill();
            mqttConnected = true;
            setAvailable(true);
            subscribeToCommands();
            
            if (stats.entityCount > 0) {
                DLOG_I(LOG_HA, "Publishing HA discovery after MQTT connect");
                publishDiscovery();
            } else {
                DLOG_W(LOG_HA, "No entities registered yet; skipping discovery on connect");
            }
        };
        on<bool>(DomoticsCore::MQTTEvents::EVENT_CONNECTED, [onMqttConnected](const bool&) { onMqttConnected(); });
        
        on<bool>(DomoticsCore::MQTTEvents::EVENT_DISCONNECTED, [this](const bool&) {
            DLOG_W(LOG_HA, "MQTT disconnected (via EventBus)");
            mqttConnected = false;
        });
        
        // Subscribe to incoming MQTT messages.
        //
        // MEM-2: ev.topic and ev.payload are already char[], and every message the
        // shared client receives arrives here — before findEntity has decided the
        // message concerns HomeAssistant at all. Wrapping them in Strings asked the
        // allocator for the topic on every single message (38 characters and up,
        // where the small-string buffer stops at 10 on ESP8266 and 14 on ESP32), to parse text the
        // component was handed as characters. They are passed through as they are.
        on<DomoticsCore::Components::MQTTMessageEvent>(DomoticsCore::MQTTEvents::EVENT_MESSAGE, [this](const DomoticsCore::Components::MQTTMessageEvent& ev) {
            handleCommand(ev.topic, ev.payload);
        });
        
        // Note: Initial MQTT state will be signaled via mqtt/connected event.
        // A re-enable (WebUI switch) gets none while the link stays up, and
        // shutdown() withdrew discovery: republish as if it had just connected.
        if (resumeConnected) {
            resumeConnected = false;
            onMqttConnected();
        }
        
        return ComponentStatus::Success;
    }
    
    void loop() override {
        // Discovery leaves one document per loop, then the states an outage
        // held a few per loop: the event queue never holds the whole burst.
        if (!mqttConnected) return;
        HAL::Platform::LockGuard guard(storeLock);
        if (discoveryPending) {
            publishNextDiscovery();
            return;
        }
        if (pendingPublishes.empty()) return;

        size_t sent = 0;
        while (sent < FLUSH_PER_LOOP && !pendingPublishes.empty()) {
            const PendingPublish& held = pendingPublishes.front();
            // A topic that cannot leave never will — a prefix or node id changed
            // under the store — so it is dropped rather than kept, or it would
            // hold every value behind it for the life of the process.
            if (!heldTopicFits(held.entity, held.attributes)) {
                heldBytes -= static_cast<uint16_t>(held.payload.length());
                pendingPublishes.erase(pendingPublishes.begin());
                stats.statesRefused++;
                continue;
            }
            // Erase only what went out: a refused publish keeps its slot, or the
            // store would lose the very value it exists to keep.
            if (!sendToBroker(held.entity, held.payload, held.attributes)) break;
            heldBytes -= static_cast<uint16_t>(held.payload.length());
            pendingPublishes.erase(pendingPublishes.begin());
            ++sent;
        }
        // Empty is where this costs nothing: erasing keeps the buffer.
        if (pendingPublishes.empty()) pendingPublishes.shrink_to_fit();
    }









    ComponentStatus shutdown() override {
        DLOG_I(LOG_HA, "Shutting down");
        // Over a dropped link nothing can be delivered: queuing it would only
        // waste EventBus space.
        resumeConnected = mqttConnected;
        if (mqttConnected) {
            setAvailable(false);
            removeDiscovery();
        }
        // begin() subscribed to the MQTT events; the WebUI switch calls
        // shutdown()/begin() directly, so drop them here or each begin() adds
        // a copy and a disabled component keeps obeying commands.
        if (__dc_eventBus) __dc_eventBus->unsubscribeOwner(this);
        mqttConnected = false;
        // While off, keep following the link only (no command handling), so a
        // drop in this window is not mistaken for "still up" by the next begin().
        // The Core removes it with the rest of this owner's subscriptions.
        on<bool>(DomoticsCore::MQTTEvents::EVENT_CONNECTED, [this](const bool&) { resumeConnected = true; });
        on<bool>(DomoticsCore::MQTTEvents::EVENT_DISCONNECTED, [this](const bool&) { resumeConnected = false; });
        // Nothing will drain it after this, and the entities it points at are
        // being withdrawn from Home Assistant anyway.
        HAL::Platform::LockGuard guard(storeLock);
        if (!pendingPublishes.empty()) {
            DLOG_W(LOG_HA, "Discarding %u held payloads at shutdown",
                   (unsigned)pendingPublishes.size());
        }
        pendingPublishes.clear();
        pendingPublishes.shrink_to_fit();
        heldBytes = 0;
        return ComponentStatus::Success;
    }





















































































































































    /**
     * @brief Publish entity state (string), or hold it until the broker is back.
     *
     * While the link is down the payload is stored, and loop() drains the store
     * after the reconnection. Safe from any task: the store takes a lock.
     */
    void publishState(const String& id, const String& state) {
        HAEntity* entity = findEntity(id);
        if (!entity) {
            DLOG_W(LOG_HA, "Entity not found: %s", id.c_str());
            return;
        }

        publishOrHold(entity, state, false);
    }





































































    /**
     * @brief Publish discovery messages for all entities
     *
     * Asynchronous: the documents leave one per loop() while the broker is
     * connected, and EVENT_DISCOVERY_PUBLISHED fires after the last one. A call
     * during a pass restarts it from the first entity.
     */
    void publishDiscovery() {
        HAL::Platform::LockGuard guard(storeLock);
        DLOG_I(LOG_HA, "Publishing discovery for %zu entities", entities.size());
        discoveryNext = 0;
        discoveryPending = true;
        stats.discoveryCount++;
    }





























































































    const HAStatistics& getStatistics() const { return stats; }



































































































































































































































































































































































































































































































































































