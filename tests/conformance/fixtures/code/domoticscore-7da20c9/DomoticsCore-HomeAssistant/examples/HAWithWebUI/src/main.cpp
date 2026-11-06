





















































































































































































    DLOG_I(LOG_APP, "System ready!");
    DLOG_I(LOG_APP, "Web interface: http://%s", HAL::WiFiHAL::getLocalIP().c_str());
    DLOG_I(LOG_APP, "MQTT Broker: %s:%d", MQTT_BROKER, MQTT_PORT);
    DLOG_I(LOG_APP, "Registered %d HA entities (5 sensors, 1 switch, 1 button)", haPtr->getStatistics().entityCount);
    DLOG_I(LOG_APP, "========================================");
}



void loop() {
    core.loop();

    // Update sensors periodically
    if (sensorTimer.isReady() && haPtr) {
        haPtr->publishState("temperature", getTemperature());
        haPtr->publishState("humidity", getHumidity());
        haPtr->publishState("uptime", (float)(HAL::Platform::getMillis() / 1000));
        haPtr->publishState("wifi_signal", (float)HAL::WiFiHAL::getRSSI());
        haPtr->publishState("free_heap", (float)HAL::Platform::getFreeHeap());

        const auto& stats = haPtr->getStatistics();
        DLOG_I(LOG_APP, "Sensors updated | States: %d, Commands: %d",
               stats.stateUpdates, stats.commandsReceived);
    }

    // Publish initial state once HA is ready
    if (!initialStatePublished && haPtr && haPtr->isReady()) {
        bool currentRelayState = HAL::Platform::digitalRead(LED_BUILTIN) == HAL::ledBuiltinOn();
        haPtr->publishState("relay", currentRelayState);
        lastRelayState = currentRelayState;
        initialStatePublished = true;
        DLOG_I(LOG_APP, "Published initial relay state: %s", currentRelayState ? "ON" : "OFF");
    }

    // Publish relay state only when it changes. No connectivity check: a state
    // published while the broker is away is held and sent when the link returns.
    if (haPtr) {
        bool currentRelayState = HAL::Platform::digitalRead(LED_BUILTIN) == HAL::ledBuiltinOn();

        if (currentRelayState != lastRelayState) {
            haPtr->publishState("relay", currentRelayState);
            DLOG_I(LOG_APP, "Relay state changed: %s", currentRelayState ? "ON" : "OFF");
            lastRelayState = currentRelayState;
        }
    }
}
