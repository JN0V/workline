


































// Runs with all of initializeAll() stacked on top of it, on the ESP8266's 4 KB
// cont stack. Keep the frame small: no large locals, buffers go in a noinline
// helper like logPromotedRecord above.
bool Core::begin(const CoreConfig& cfg) {
    if (initialized) {
        DLOG_W(LOG_CORE, "Core already initialized");
        return true;
    }
    
    // OBS-3: read what the last death left in RTC before anything else runs.
    // Idempotent: System::begin() may already have called it with the hold.
    FlightRecorder& recorder = FlightRecorder::instance();
    recorder.begin(false);

    config = cfg;
    
    // Initialize Serial if not already done
    if (!HAL::isLoggerReady()) {
        HAL::initializeLogging(115200);
    }

#ifdef DOMOTICS_BUILD_ID
    DLOG_I(LOG_CORE, "Build id: %s", DOMOTICS_BUILD_ID);
#endif
    if (!recorder.restartHookInstalled()) {
        DLOG_W(LOG_CORE, "Restart hook not registered: an esp_restart() outside the HAL will read as unexplained");
    }
    if (HAL::Platform::supportsFailedAllocHook() && !recorder.failedAllocHookInstalled()) {
        DLOG_W(LOG_CORE, "Failed-allocation hook not registered: survived allocation failures will not be recorded");
    }
    if (!DOMOTICS_FLIGHT_RECORDER_TICK && !HAL::Platform::supportsFailedAllocHook()) {
        DLOG_W(LOG_CORE, "Sampler compiled out: survived allocation failures will not be recorded on this platform");
    }
    if (recorder.hasPromotedRecord()) {
        logPromotedRecord(recorder);
        // A bare Core has no Storage: the log line above is where the record
        // got out, so the fresh record can take RTC now. System persists
        // first and acknowledges itself.
        if (!recorder.acknowledgementDeferred()) recorder.acknowledge();
    }
    
    // Generate unique device ID if not provided
    if (config.deviceId.isEmpty()) {
        config.deviceId = "DC" + HAL::formatChipIdHex();
        config.deviceId = HAL::toUpperCase(config.deviceId);
    }
    
    DLOG_I(LOG_CORE, "DomoticsCore initializing...");
    DLOG_I(LOG_CORE, "Device: %s (ID: %s)", config.deviceName.c_str(), config.deviceId.c_str());

    // Detect memory profile BEFORE component initialization
    // Components can query MemoryManager::instance().getProfile() during their begin()
    auto& memMgr = MemoryManager::instance();
    memMgr.detectProfile();
    DLOG_I(LOG_CORE, "Memory profile: %s (heap: %u bytes)",
           memMgr.getProfileName(), memMgr.getHeapAtBoot());

    // Provide Core reference to registry for component injection
    componentRegistry.setCore(this);

    // Initialize all registered components
    Components::ComponentStatus status = componentRegistry.initializeAll();
    if (status != Components::ComponentStatus::Success) {
        DLOG_E(LOG_CORE, "Failed to initialize components: %s", Components::statusToString(status));
        return false;
    }
    
    initialized = true;
    DLOG_I(LOG_CORE, "Core initialization complete");
    return true;
}

void Core::loop() {
    if (!initialized) return;
    
    // Run component loops
    componentRegistry.loopAll();

    // OBS-3: heap running minimum every loop, a sample to RTC every 10 s.
    FlightRecorder& recorder = FlightRecorder::instance();
    const uint32_t drops = componentRegistry.getEventBus().getDroppedCount();
    recorder.noteEventDrops(drops);
    recorder.tick();

    // LO-5 / BUG-36: the queue drops silently; say so, at most once a minute.
    if (drops != lastDropsLogged_ && (lastDropLog_ == 0 || HAL::getMillis() - lastDropLog_ >= 60000)) {
        DLOG_W(LOG_CORE, "EventBus dropped %lu events (queue budget %u B, peak %u %%)",
               (unsigned long)drops, (unsigned)DomoticsCore::Utils::QueueCost::kBudgetBytes,
               (unsigned)componentRegistry.getEventBus().getQueueHighWaterPct());
        lastDropsLogged_ = drops;
        lastDropLog_ = HAL::getMillis();
    }

    // OBS-4: an allocation failed and the firmware survived it; say so, at
    // most once a minute. The recorder itself stays log-free (crash path).
    uint32_t fails = 0, lastFailSize = 0;
    recorder.failedAllocSnapshot(fails, lastFailSize);
    if (fails != lastFailsLogged_ && (lastFailLog_ == 0 || HAL::getMillis() - lastFailLog_ >= 60000)) {
        DLOG_W(LOG_CORE, "Allocation failures since boot: %lu, last %lu B",
               (unsigned long)fails, (unsigned long)lastFailSize);
        lastFailsLogged_ = fails;
        lastFailLog_ = HAL::getMillis();
    }
    
    // Core minimal heartbeat
    static unsigned long lastHeartbeat = 0;
    if (HAL::getMillis() - lastHeartbeat >= 60000) { // Every minute
        lastHeartbeat = HAL::getMillis();
        DLOG_D(LOG_CORE, "Core heartbeat - uptime: %lu seconds, components: %zu", 
               HAL::getMillis() / 1000, componentRegistry.getComponentCount());
    }
}

void Core::shutdown() {
    if (!initialized) return;
    
    DLOG_I(LOG_CORE, "Shutting down DomoticsCore");
    // Shutdown all components first
    componentRegistry.shutdownAll();
    
    initialized = false;
    DLOG_I(LOG_CORE, "Core shutdown complete");
}


