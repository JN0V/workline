























































































































/** @brief Configure the publisher from the MQTT client id, give it its sink, and publish the crash record at every connect. */
inline void setupTelemetry(Core& core, const SystemConfig& config, SystemTelemetry& t,
                           CrashSummary& lastDeath, Components::MQTTComponent* mqttComp) {
    if (config.telemetryIntervalSec == 0) {
        DLOG_I(LOG_TELEMETRY, "Telemetry off (telemetryIntervalSec = 0)");
        return;
    }
    if (!t.configure(mqttComp->getConfig().clientId.c_str(), config.telemetryIntervalSec, config.telemetryHeapFloor)) {
        DLOG_W(LOG_TELEMETRY, "Telemetry off: MQTT client id longer than %u characters or carrying a wildcard",
               (unsigned)SystemTelemetry::CLIENT_ID_MAX);
        return;
    }
    // The tick never queues (a deferred sample is a stale one); the crash
    // publish at connect may, it is retained and lands after HA's burst.
    t.setSink([mqttComp](const char* topic, const char* payload, size_t len, bool retain, bool mayQueue) {
        if (mayQueue) return mqttComp->publish(String(topic), String(payload), 0, retain);
        return mqttComp->publishNow(topic, payload, len, retain);
    });
    // mqttComp->connect() may already have run in this same setup: the
    // connect event is only enqueued by it, dispatched on a later loop(), so
    // a subscription made here still sees it.
    core.getEventBus().subscribe(MQTTEvents::EVENT_CONNECTED, [&t, &lastDeath](const void* payload) {
        if (!payload || !*static_cast<const bool*>(payload)) return;
        refreshCoreDumpIn(lastDeath);
        if (!t.publishCrash(lastDeath)) {
            DLOG_W(LOG_TELEMETRY, "Crash record not published on %s", t.crashTopic());
        }
    });
    DLOG_I(LOG_TELEMETRY, "Telemetry every %u s on %s; last death on %s",
           (unsigned)config.telemetryIntervalSec, t.telemetryTopic(), t.crashTopic());
}

















































