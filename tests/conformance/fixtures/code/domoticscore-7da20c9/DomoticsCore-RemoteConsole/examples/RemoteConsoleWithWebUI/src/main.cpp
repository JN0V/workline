

































void setup() {
    // Initialize logging
    HAL::Platform::initializeLogging();

    DLOG_I(LOG_APP, "========================================");
    DLOG_I(LOG_APP, "DomoticsCore - RemoteConsole + WebUI");
    DLOG_I(LOG_APP, "========================================");

    // Connect to WiFi using HAL
    DLOG_I(LOG_APP, "Connecting to WiFi: %s", WIFI_SSID);
    HAL::WiFiHAL::init();
    HAL::WiFiHAL::setMode(HAL::WiFiHAL::Mode::Station);
    HAL::WiFiHAL::connect(WIFI_SSID, WIFI_PASSWORD);

    int attempts = 0;
    while (!HAL::WiFiHAL::isConnected() && attempts < 40) {
        HAL::Platform::delayMs(500);
        attempts++;
    }

    if (!HAL::WiFiHAL::isConnected()) {
        DLOG_W(LOG_APP, "WiFi STA connection failed, starting AP mode...");
        HAL::WiFiHAL::setMode(HAL::WiFiHAL::Mode::AccessPoint);
        HAL::WiFiHAL::startAP(AP_SSID, AP_PASSWORD);
        DLOG_I(LOG_APP, "AP started: %s", AP_SSID);
    } else {
        DLOG_I(LOG_APP, "WiFi connected: %s", HAL::WiFiHAL::getLocalIP().c_str());
    }

    // Configure WebUI
    auto webui = std::make_unique<WebUIComponent>();

    // Configure RemoteConsole
    RemoteConsoleConfig consoleConfig;
    consoleConfig.enabled = true;
    consoleConfig.port = 23;  // Standard telnet port
    consoleConfig.maxClients = 3;  // Allow up to 3 concurrent telnet clients
    consoleConfig.colorOutput = true;  // ANSI colors in telnet
    consoleConfig.allowCommands = true;  // Enable built-in commands
    consoleConfig.defaultLogLevel = LOG_LEVEL_INFO;

    auto console = std::make_unique<RemoteConsoleComponent>(consoleConfig);

    // Add components to core
    core.addComponent(std::move(webui));
    core.addComponent(std::move(console));

    // Initialize
    core.begin();

    // Register WebUI provider for RemoteConsole
    auto* webuiPtr = core.getComponent<WebUIComponent>("WebUI");
    auto* consolePtr = core.getComponent<RemoteConsoleComponent>("RemoteConsole");
    if (webuiPtr && consolePtr) {
        auto* provider = new DomoticsCore::Components::WebUI::RemoteConsoleWebUI(consolePtr);
        webuiPtr->registerProviderWithComponent(provider, consolePtr);
        provider->init(webuiPtr);
    }

    DLOG_I(LOG_APP, "========================================");
    DLOG_I(LOG_APP, "System ready!");
    DLOG_I(LOG_APP, "Telnet: %s:%d", HAL::WiFiHAL::getLocalIP().c_str(), consoleConfig.port);
    DLOG_I(LOG_APP, "WebUI: http://%s", HAL::WiFiHAL::getLocalIP().c_str());
    DLOG_I(LOG_APP, "========================================");
}















