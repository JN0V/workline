


























































void setup() {
    HAL::Platform::initializeLogging();
    
    DLOG_I(LOG_APP, "========================================");
    DLOG_I(LOG_APP, "DomoticsCore - RemoteConsole Example");
    DLOG_I(LOG_APP, "========================================");
    
    // Connect to WiFi using HAL
    DLOG_I(LOG_APP, "Connecting to WiFi: %s", WIFI_SSID);
    HAL::WiFiHAL::init();
    HAL::WiFiHAL::setMode(HAL::WiFiHAL::Mode::Station);
    HAL::WiFiHAL::connect(WIFI_SSID, WIFI_PASSWORD);
    
    int attempts = 0;
    while (!HAL::WiFiHAL::isConnected() && attempts < 40) {
        HAL::Platform::delayMs(500);
        DLOG_D(LOG_APP, ".");
        attempts++;
    }
    
    if (!HAL::WiFiHAL::isConnected()) {
        DLOG_E(LOG_APP, "WiFi connection failed!");
        while (1) HAL::Platform::delayMs(1000);
    }
    
    DLOG_I(LOG_APP, "WiFi connected: %s", HAL::WiFiHAL::getLocalIP().c_str());
    
    // Configure RemoteConsole
    RemoteConsoleConfig config;
    config.enabled = true;
    config.port = 23;
    config.colorOutput = true;
    config.defaultLogLevel = LOG_LEVEL_INFO;
    
    auto console = std::make_unique<RemoteConsoleComponent>(config);
    auto* consolePtr = console.get();
    
    // Register custom command
    consolePtr->registerCommand("test", [](const String& args) {
        return "Test command executed with args: " + args + "\n";
    });
    
    consolePtr->registerCommand("sensors", [](const String& args) {
        String result = "\nSensor Values:\n";
        result += "  Temperature: 22.5°C\n";
        result += "  Humidity: 45%\n";
        result += "  Pressure: 1013 hPa\n";
        return result;
    });
    
    // Add to core
    core.addComponent(std::move(console));
    
    // Initialize
    if (!core.begin()) {
        DLOG_E(LOG_APP, "Failed to initialize core!");
        while (1) HAL::Platform::delayMs(1000);
    }
    
    DLOG_I(LOG_APP, "========================================");
    DLOG_I(LOG_APP, "System ready!");
    DLOG_I(LOG_APP, "Telnet: %s:%d", HAL::WiFiHAL::getLocalIP().c_str(), config.port);
    DLOG_I(LOG_APP, "========================================");
}






















