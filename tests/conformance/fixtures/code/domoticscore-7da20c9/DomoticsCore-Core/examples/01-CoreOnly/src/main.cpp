












void setup() {
    // Initialize Serial early for logging before core initialization
    HAL::initializeLogging(115200);

    // ============================================================================
    // EXAMPLE 01: Core Only
    // ============================================================================
    // This example demonstrates the basic DomoticsCore framework:
    // - Core initialization with custom device configuration
    // - Platform HAL integration (chip info, memory, temperature)
    // - Non-blocking timer patterns (10s heartbeat, 30s status)
    // Expected: Device info logs, regular heartbeat and status reports
    // ============================================================================
    
    DLOG_I(LOG_APP, "=== Core Only Example ===");
    DLOG_I(LOG_APP, "Basic DomoticsCore framework demonstration");
    DLOG_I(LOG_APP, "Heartbeat every 10s, Status every 30s");
    DLOG_I(LOG_APP, "=========================");

    // Create core with custom device name
    CoreConfig config;
    config.deviceName = "MyESP32Device";
    config.logLevel = 3; // INFO level
    
    // Core initialized
    
    if (!core.begin(config)) {
        DLOG_E(LOG_APP, "Failed to initialize core!");
        return;
    }
    
    DLOG_I(LOG_APP, "Device configured: %s (ID: %s)", 
           core.getDeviceName().c_str(), 
           core.getDeviceId().c_str());
    DLOG_I(LOG_APP, "Setup complete - device ready");
}

void loop() {
    core.loop();
    
    // Non-blocking heartbeat every 10 seconds
    if (heartbeatTimer.isReady()) {
        DLOG_I(LOG_APP, "Heartbeat - uptime: %lu seconds", millis() / 1000);
    }
    
    // Non-blocking status report every 30 seconds
    if (statusTimer.isReady()) {
        DLOG_I(LOG_SYSTEM, "Free heap: %lu bytes", (unsigned long)HAL::Platform::getFreeHeap());
        float temp = HAL::Platform::getTemperature();
        if (!isnan(temp)) {
            DLOG_I(LOG_SYSTEM, "Chip temperature: %.1f°C", temp);
        }
    }
    
    // No blocking delay needed - timers handle everything
}
