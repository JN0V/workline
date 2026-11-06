#pragma once

/**
 * @file RemoteConsole.h
 * @brief Telnet-based remote console for log streaming and command execution
 */

#include <DomoticsCore/AuthDelay.h>
#include <DomoticsCore/StringParse.h>
#include <DomoticsCore/IComponent.h>
#include <DomoticsCore/Logger.h>
#include <DomoticsCore/CoreLog_HAL.h>
#include <DomoticsCore/Platform_HAL.h>    // For restart()
#include <DomoticsCore/Wifi_HAL.h>        // For WiFi functions
#include <DomoticsCore/WiFiServer_HAL.h>  // For WiFiServer and WiFiClient
// Platform_HAL.h provides: getFreeHeap(), getChipModel(), getChipRevision(), getCpuFreqMHz()
#include <new>      // std::nothrow
#include <vector>
#include <map>
#include <functional>

namespace DomoticsCore {
namespace Components {

// Log entry structure - uses String for compact storage
struct LogEntry {
    uint32_t timestamp;
    LogLevel level;
    String tag;
    String message;
    
    LogEntry() : timestamp(0), level(LOG_LEVEL_INFO) {}
    LogEntry(uint32_t ts, LogLevel lvl, const char* t, const char* msg)
        : timestamp(ts), level(lvl), tag(t), message(msg) {}
};

// Remote console configuration
struct RemoteConsoleConfig {
    bool enabled = true;
    uint16_t port = 23;                    // Telnet port
    bool requireAuth = false;              // Password authentication
    String password = "";                  // Auth password
    uint32_t bufferSize = DOMOTICS_LOG_BUFFER_SIZE;  // Lines of history: ESP8266 20, ESP32 64
    bool allowCommands = true;             // Enable command execution
    uint32_t authTimeoutMs = 10000;        // Auth timeout (10s default, 0 = no timeout)
    uint32_t authDelayMaxMs = 8000;        // SEC-4: cap of the doubling wait before the next auth attempt is read (0 = none)
    std::vector<HAL::IPAddress> allowedIPs;     // IP whitelist (empty = all allowed)
    bool colorOutput = true;               // ANSI color codes
    uint32_t maxClients = 3;               // Max concurrent connections
    LogLevel defaultLogLevel = LOG_LEVEL_INFO;  // Initial log level
};

// Command handler function type
typedef std::function<String(const String& args)> CommandHandler;

/**
 * @class RemoteConsoleComponent
 * @brief Telnet server for remote log viewing and command execution
 * 
 * Features:
 * - Real-time log streaming via Telnet
 * - Circular log buffer with configurable size
 * - Runtime log level control
 * - Command processor with extensible commands
 * - ANSI color-coded output
 * - Password authentication
 * - IP whitelist support
 */
class RemoteConsoleComponent : public IComponent {
private:
    RemoteConsoleConfig config;
    HAL::WiFiServer* telnetServer = nullptr;
    LoggerCallbacks::CallbackId loggerCallbackId_ = 0;
    bool sdkOutputEnabled_ = false;   // off is the default: see the SDK note in CoreLog_ESP8266.h
    bool serverWanted_ = false;       // begin() wants one, shutdown() does not: loop() reads this
    uint32_t coreLogDroppedReported_ = 0;
    uint32_t coreLogDropReportedAt_ = 0;
    bool coreLogDropEverReported_ = false;   // millis() == 0 is a timestamp, not a sentinel
    uint32_t nextClientId = 1;
    std::vector<std::pair<uint32_t, HAL::WiFiClient>> clients;
    
    // Circular buffer for log entries - grows lazily to avoid OOM on startup
    std::vector<LogEntry> logBuffer;
    size_t logBufferHead = 0;    // Next write position
    size_t logBufferCount = 0;   // Current number of entries
    
    std::map<String, CommandHandler> commands;
    // One record per connected client (key = client ID). One map, so every
    // erase site is one line and a per-client field cannot be left behind (MEM-1).
    struct ClientState {
        bool authenticated = false;
        unsigned long connectedAt = 0;
        String buffer;                     // command bytes received so far
        unsigned long authNotBefore = 0;   // SEC-4: an auth line is held until this
        bool authPending = false;
        String authPendingArgs;
    };
    std::map<uint32_t, ClientState> clientState;
    // Per-address failure memory, so a reconnection keeps its wait.
    Utils::AuthDelay authDelay;
    LogLevel currentLogLevel;
    std::vector<String> tagFilter;  // Empty = show all
    bool connectionInfoDisplayed = false;  // Track if we've shown connection info
    bool rebootPending = false;
    unsigned long rebootRequestedAt = 0;

public:
    RemoteConsoleComponent(const RemoteConsoleConfig& cfg = RemoteConsoleConfig())
        : config(cfg), currentLogLevel(cfg.defaultLogLevel) {
        
        metadata.name = "RemoteConsole";
        metadata.version = "1.7.3";  // x-release-please-version
        metadata.author = "DomoticsCore";
        metadata.description = "Telnet-based remote console with log streaming";
        metadata.category = "Debug";
        metadata.tags = {"telnet", "console", "debug", "logging"};
        
        // Register built-in commands
        registerBuiltInCommands();
    }

    uint16_t getPort() const { return config.port; }
    HAL::WiFiServer* getServer() const { return telnetServer; }
    size_t clientStateEntriesForTest() const { return clientState.size(); }
    size_t logBufferCapacityForTest() const { return logBuffer.capacity(); }

    LogLevel getLogLevel() const { return currentLogLevel; }

    bool setPort(uint16_t port) {
        if (port == 0) return false;
        if (config.port == port) return true;

        config.port = port;

        if (!config.enabled) {
            return true;
        }

        if (!telnetServer) {
            return true;
        }

        for (auto& entry : clients) {
            auto& client = entry.second;
            if (client.connected()) {
                client.println("\nRemoteConsole port changed - disconnecting...");
                client.stop();
            }
        }
        clients.clear();
        clientState.clear();
        clients.shrink_to_fit();

        telnetServer->stop();
        delete telnetServer;
        telnetServer = nullptr;

        openServer("restarted");
        return true;
    }
    
    ~RemoteConsoleComponent() {
        if (telnetServer) {
            delete telnetServer;
        }
    }
    
    
    ComponentStatus begin() override {
        if (!config.enabled) {
            DLOG_I(LOG_CONSOLE, "RemoteConsole disabled in config");
            setStatus(ComponentStatus::Success);
            return ComponentStatus::Success;
        }
        
        if (config.requireAuth && config.password.isEmpty()) {
            DLOG_W(LOG_CONSOLE, "requireAuth with an empty password: authentication disabled");  // SEC-14
            config.requireAuth = false;
        }

        // Register logger callback
        loggerCallbackId_ = LoggerCallbacks::addCallback([this](LogLevel level, const char* tag, const char* msg) {
            this->log(level, tag, msg);
        });
        
        // The lines the platform writes itself — the Arduino core's, ESP-IDF's —
        // never reached a client: they go to the UART, which a device in the
        // field does not have. loop() drains them out of the HAL's intake.
        if (HAL::CoreLog::installCapture()) {
            DLOG_I(LOG_CONSOLE, "Core log capture installed (%u slots)", (unsigned)HAL::CoreLog::SLOTS);
        }

        // A listening socket needs an IP stack, which ESP8266 has from boot and
        // ESP32 gets with its first network interface. Opening before that stops
        // the firmware inside lwIP, so a console that starts first waits for one.
        serverWanted_ = true;
        openServer();
        if (!telnetServer) {
            DLOG_I(LOG_CONSOLE, "RemoteConsole waiting for a network stack to listen on port %d",
                   config.port);
        }
        
        setStatus(ComponentStatus::Success);
        return ComponentStatus::Success;
    }
    
    void onComponentsReady(const ComponentRegistry& /*registry*/) override {
        // Display connection info once WiFi is connected
        displayConnectionInfo();
    }
    
    void loop() override {
        // Non-blocking reboot (R9 — must be before status guard)
        if (rebootPending && (HAL::Platform::getMillis() - rebootRequestedAt >= 100)) {
            rebootPending = false;
            HAL::restart();
        }

        // Before the client work: a line the platform wrote is worth as much as
        // one of ours, and it has been waiting in a buffer that cannot grow.
        drainCoreLog();

        // A stack that appeared after begin() — the radio coming up, or another
        // transport — is when a deferred server opens. Never after shutdown():
        // a console disabled at runtime keeps its port closed.
        if (!telnetServer && serverWanted_) openServer();

        if (getLastStatus() != ComponentStatus::Success || !telnetServer) return;
        
        // Check if WiFi connected and we haven't displayed info yet
        if (!connectionInfoDisplayed && HAL::WiFiHAL::isConnected()) {
            displayConnectionInfo();
        }
        
        // Accept new clients (Task 28)
        if (telnetServer->hasClient()) {
            HAL::WiFiClient newClient = telnetServer->accept();

            if (newClient) {
                // Check max clients
                if (clients.size() >= config.maxClients) {
                    newClient.println("Max clients reached. Disconnecting.");
                    newClient.stop();
                } else if (!isIPAllowed(newClient.remoteIP())) {
                    newClient.println("IP not allowed. Disconnecting.");
                    newClient.stop();
                } else {
                    uint32_t clientId = nextClientId++;
                    clients.push_back({clientId, newClient});
                    const unsigned long now = HAL::Platform::getMillis();
                    ClientState& st = clientState[clientId];
                    st.authenticated = !config.requireAuth;
                    st.connectedAt = now;
                    st.authNotBefore = authDelay.notBefore((uint32_t)clients.back().second.remoteIP(), now, config.authDelayMaxMs);

                    DLOG_I(LOG_CONSOLE, "Client connected: #%u", clientId);

                    sendWelcome(clients.back().second);
                }
            }
        }

        // Auth timeout pass — separate from handleClient to avoid processing partially-disconnected clients
        if (config.requireAuth && config.authTimeoutMs > 0) {
            unsigned long now = HAL::Platform::getMillis();
            for (auto it = clients.begin(); it != clients.end(); ) {
                uint32_t cid = it->first;
                HAL::WiFiClient& client = it->second;
                const ClientState& st = clientState[cid];
                if (!st.authenticated && (now - st.connectedAt) >= config.authTimeoutMs) {
                    client.println("Authentication timeout. Disconnecting.");
                    client.stop();
                    clientState.erase(cid);
                    it = clients.erase(it);
                } else {
                    ++it;
                }
            }
        }

        // Handle existing clients
        bool erased = false;
        for (auto it = clients.begin(); it != clients.end(); ) {
            uint32_t cid = it->first;
            HAL::WiFiClient& client = it->second;
            if (!client.connected()) {
                clientState.erase(cid);

                DLOG_I(LOG_CONSOLE, "Client disconnected: #%u", cid);
                it = clients.erase(it);
                erased = true;
            } else {
                handleClient(cid, client);
                ++it;
            }
        }
        if (erased) clients.shrink_to_fit();
    }
    
    ComponentStatus shutdown() override {
        LoggerCallbacks::removeCallback(loggerCallbackId_);
        HAL::CoreLog::removeCapture();   // a no-op when begin() never took a sink
        if (telnetServer) {
            // Disconnect all clients
            for (auto& entry : clients) {
                entry.second.println("\nRemoteConsole shutting down...");
                entry.second.stop();
            }
            clients.clear();
            clientState.clear();
            clients.shrink_to_fit();

            telnetServer->stop();
            delete telnetServer;
            telnetServer = nullptr;
        }
        
        serverWanted_ = false;
        DLOG_I(LOG_CONSOLE, "RemoteConsole shut down");
        setStatus(ComponentStatus::Success);
        return ComponentStatus::Success;
    }
    
    // At most this many platform lines per loop: a boot burst must not hold the
    // loop, and what it leaves behind is drained on the next pass.
    static constexpr size_t CORE_LOG_PER_LOOP = 4;

    /**
     * @brief Move what the platform logged into the console's own buffer.
     *
     * The level is read from the shape of the line, so an error stays greppable
     * as one: ESP-IDF writes "E (1591) gpio: …" and the Arduino core writes
     * "[  1638][E][Preferences.cpp:50] begin(): …". Anything else is information.
     */
    void drainCoreLog() {
        if (!HAL::CoreLog::captureInstalled()) return;   // nothing takes the lock for nothing

        char line[HAL::CoreLog::MAX_LINE];
        for (size_t i = 0; i < CORE_LOG_PER_LOOP; ++i) {
            if (HAL::CoreLog::drainLine(line, sizeof(line)) == 0) break;
            log(coreLogLevelOf(line), LOG_PLATFORM, line);
        }

        // A burst the intake could not hold is said once a minute, the way the
        // event queue says its own drops. A reinstall zeroes the counter, so the
        // delta is clamped rather than wrapped through four billion.
        const uint32_t dropped = HAL::CoreLog::droppedLines();
        if (dropped < coreLogDroppedReported_) coreLogDroppedReported_ = 0;
        const uint32_t now = HAL::Platform::getMillis();
        if (dropped != coreLogDroppedReported_ &&
            (!coreLogDropEverReported_ || now - coreLogDropReportedAt_ >= 60000)) {
            DLOG_W(LOG_CONSOLE, "Core log intake dropped %u lines (%u slots)",
                   (unsigned)(dropped - coreLogDroppedReported_), (unsigned)HAL::CoreLog::SLOTS);
            coreLogDroppedReported_ = dropped;
            coreLogDropReportedAt_ = now;
            coreLogDropEverReported_ = true;
        }
    }

    /** @brief The level a platform line carries, read from its two known shapes. */
    static LogLevel coreLogLevelOf(const char* line) {
        char letter = '\0';
        // With the core's log colours on, every line opens with an escape
        // sequence; the shape that carries the level is behind it.
        if (line[0] == '\033') {
            const char* end = line;
            while (*end && *end != 'm') ++end;
            if (*end == 'm') line = end + 1;
        }
        if (line[0] && line[1] == ' ' && line[2] == '(') {
            letter = line[0];                      // ESP-IDF: "E (1591) tag: …"
        } else if (line[0] == '[') {
            const char* mark = strstr(line, "][");
            if (mark && mark[2] && mark[3] == ']') letter = mark[2];   // ARDUHAL: "[  12][E][…"
        }
        switch (letter) {
            case 'E': return LOG_LEVEL_ERROR;
            case 'W': return LOG_LEVEL_WARN;
            case 'D': return LOG_LEVEL_DEBUG;
            case 'V': return LOG_LEVEL_VERBOSE;
            default:  return LOG_LEVEL_INFO;
        }
    }

    /**
     * @brief Log a message to the buffer and connected clients
     */
    void log(LogLevel level, const char* tag, const char* message) {
        if (level > currentLogLevel) return;
        
        // Check tag filter
        if (!tagFilter.empty()) {
            bool tagMatch = false;
            for (const auto& filter : tagFilter) {
                if (filter == tag) {
                    tagMatch = true;
                    break;
                }
            }
            if (!tagMatch) return;
        }
        
        LogEntry entry(HAL::Platform::getMillis(), level, tag, message);

        // Clients first: the entry is moved into the history below.
        if (!clients.empty()) {
            String formatted = formatLogEntry(entry);
            for (auto& peer : clients) {
                auto& client = peer.second;
                auto st = clientState.find(peer.first);
                if (client.connected() && st != clientState.end() && st->second.authenticated) {
                    client.print(formatted);
                }
            }
        }

        if (config.bufferSize > 0) {
            if (logBufferCount < config.bufferSize) {
                // One allocation of exactly the cap: growing by doubling leaves dead
                // slots at most sizes and reallocates the whole vector on the way.
                if (logBuffer.capacity() < config.bufferSize) logBuffer.reserve(config.bufferSize);
                logBuffer.push_back(std::move(entry));
                logBufferCount++;
            } else {
                // Destroyed, then rebuilt: assigning, even by move, lets an ESP32 String
                // keep a larger buffer than the new line needs, so every slot would
                // ratchet up to the longest line it ever held.
                LogEntry* slot = &logBuffer[logBufferHead];
                slot->~LogEntry();
                new (slot) LogEntry(std::move(entry));
            }
            logBufferHead = (logBufferHead + 1) % config.bufferSize;
        }
    }
    
    /**
     * @brief Register a custom command
     */
    void registerCommand(const String& cmd, CommandHandler handler) {
        commands[cmd] = handler;
        DLOG_D(LOG_CONSOLE, "Registered command: %s", cmd.c_str());
    }
    
    /**
     * @brief Set runtime log level
     */
    void setLogLevel(LogLevel level) {
        currentLogLevel = level;
        DLOG_I(LOG_CONSOLE, "Log level set to: %d", level);
    }
    
    /**
     * @brief Set tag filter (empty = show all)
     */
    void setTagFilter(const std::vector<String>& tags) {
        tagFilter = tags;
    }
    
    /**
     * @brief Clear log buffer and release memory
     */
    void clearBuffer() {
        logBuffer.clear();
        logBuffer.shrink_to_fit();  // Release memory back to heap
        logBufferHead = 0;
        logBufferCount = 0;
        DLOG_I(LOG_CONSOLE, "Log buffer cleared");
    }
    
    /**
     * @brief Get recent logs from circular buffer
     */
    std::vector<LogEntry> getRecentLogs(uint32_t count = 100) {
        std::vector<LogEntry> result;
        if (logBufferCount == 0 || config.bufferSize == 0) return result;
        
        uint32_t actualCount = (count < logBufferCount) ? count : logBufferCount;
        result.reserve(actualCount);
        
        // Calculate start position in circular buffer
        size_t startIdx;
        if (logBufferCount < config.bufferSize) {
            // Buffer not full yet - start from beginning
            startIdx = (logBufferCount > actualCount) ? (logBufferCount - actualCount) : 0;
        } else {
            // Buffer is full - oldest is at logBufferHead
            startIdx = (logBufferHead + config.bufferSize - actualCount) % config.bufferSize;
        }
        
        for (uint32_t i = 0; i < actualCount; i++) {
            size_t idx = (startIdx + i) % config.bufferSize;
            result.push_back(logBuffer[idx]);
        }
        
        return result;
    }

private:
    /**
     * @brief Open the telnet server, when the platform says a socket can be.
     *
     * Self-guarded: a caller that asks too early is deferred, not aborted, and
     * loop() tries again. An allocation that fails leaves the pointer null, so
     * the same retry covers it.
     */
    void openServer(const char* what = "started") {
        if (telnetServer || !HAL::canOpenServer()) return;
        telnetServer = new (std::nothrow) HAL::WiFiServer(config.port);
        if (!telnetServer) {
            DLOG_E(LOG_CONSOLE, "No heap for the telnet server; will try again");
            return;
        }
        telnetServer->begin();
        telnetServer->setNoDelay(true);
        DLOG_I(LOG_CONSOLE, "RemoteConsole %s on port %d", what, config.port);
    }

    void registerBuiltInCommands() {
        // Help command
        registerCommand("help", [this](const String& args) {
            // MEM-2. The constant part used to be ten run-time appends of
            // compile-time literals — ten length scans and up to ten growth
            // steps, since String rounds capacity to a 16-byte multiple
            // (WString.cpp:229) and this text is 427 characters. Adjacent
            // string literals are concatenated by the compiler, so the text
            // below is one stored constant and one copy, byte for byte what
            // the ten appends produced.
            //
            // Not measured on a board, and deliberately so:
            // registerBuiltInCommands() is private, this lambda is reachable
            // only through the telnet read loop, and DomoticsCore-RemoteConsole
            // declares no ESP8266 environment. Measuring it would mean
            // inventing public API and a device suite for a string built once
            // per typed command.
            static const char HELP_TEXT[] =
                "\nAvailable commands:\n"
                "  help              - Show this help\n"
                "  clear             - Clear log buffer\n"
                "  level <level>     - Set log level (0-5: NONE/ERROR/WARN/INFO/DEBUG/VERBOSE)\n"
                "  filter <tag>      - Filter logs by tag (empty = show all)\n"
                "  info              - System information\n"
                "  heap              - Memory usage\n"
                "  core [on|off]     - The platform's own log lines: state, or the SDK narration\n"
                "  reboot            - Restart device\n"
                "  auth <password>   - Authenticate (if auth required)\n"
                "  quit              - Disconnect\n";

            String help = HELP_TEXT;

            // Add custom commands
            for (const auto& cmd : commands) {
                if (cmd.first != "help" && cmd.first != "clear" && 
                    cmd.first != "level" && cmd.first != "filter" &&
                    cmd.first != "info" && cmd.first != "heap" && 
                    cmd.first != "reboot" && cmd.first != "quit" &&
                    cmd.first != "core") {
                    help += "  " + cmd.first + "\n";
                }
            }
            
            return help;
        });
        
        // The platform's own lines: what is captured, and — where the platform
        // has a switch for it — whether its SDK narrates at all.
        registerCommand("core", [this](const String& args) -> String {
            char buf[224];
            const bool wantsChange = !args.isEmpty();
            const bool enable = args == "on";
            if (wantsChange && !enable && args != "off") {
                return "Usage: core [on|off]\n";
            }
            if (wantsChange) {
                if (!HAL::CoreLog::supportsSdkOutputSwitch()) {
                    return "This platform has nothing to switch: its core and SDK lines are already captured\n";
                }
                HAL::CoreLog::setSdkOutput(enable);
                sdkOutputEnabled_ = enable;
                if (enable) {
                    return "SDK narration on. It reaches the serial port as well, and costs about a "
                           "line a second while a join fails\n";
                }
                return "SDK narration off\n";
            }
            snprintf(buf, sizeof(buf),
                     "Platform lines: %s, %u intake slots, %u dropped, %u ours ignored\n"
                     "SDK narration: %s\n",
                     HAL::CoreLog::captureInstalled() ? "captured" : "not captured",
                     (unsigned)HAL::CoreLog::SLOTS, (unsigned)HAL::CoreLog::droppedLines(),
                     (unsigned)HAL::CoreLog::ignoredLines(),
                     HAL::CoreLog::supportsSdkOutputSwitch()
                         ? (sdkOutputEnabled_ ? "on (set here)" : "off (set here)")
                         : "not a switch on this platform");
            return String(buf);
        });

        // Clear command
        registerCommand("clear", [this](const String& args) {
            clearBuffer();
            return "Log buffer cleared\n";
        });
        
        // Level command
        registerCommand("level", [this](const String& args) -> String {
            char buf[128];
            if (args.isEmpty()) {
                snprintf(buf, sizeof(buf), "Current log level: %d\n", (int)currentLogLevel);
                return String(buf);
            }

            // Digits only, then the range: toInt() is atol(), which reads "3x"
            // as 3 and "abc" as 0 — LOG_LEVEL_NONE, silencing the log on a typo.
            // The range is the Logger's own, the one the WebUI route lists.
            if (!Utils::digitsOnly(args) || args.toInt() > LOG_LEVEL_VERBOSE) {
                return String("Invalid level. Use 0-5 (NONE/ERROR/WARN/INFO/DEBUG/VERBOSE)\n");
            }
            int level = args.toInt();

            setLogLevel((LogLevel)level);
            snprintf(buf, sizeof(buf), "Log level set to: %d\n", level);
            return String(buf);
        });
        
        // Filter command
        registerCommand("filter", [this](const String& args) -> String {
            if (args.isEmpty()) {
                tagFilter.clear();
                return String("Tag filter cleared (showing all)\n");
            }

            tagFilter.clear();
            tagFilter.push_back(args);
            char buf[128];
            snprintf(buf, sizeof(buf), "Filtering logs by tag: %s\n", args.c_str());
            return String(buf);
        });
        
        // Info command
        registerCommand("info", [this](const String& args) {
            char buf[384];
            snprintf(buf, sizeof(buf),
                     "\nSystem Information:\n"
                     "  Uptime: %lus\n"
                     "  Free Heap: %lu bytes\n"
                     "  Chip: %s Rev%d\n"
                     "  CPU Freq: %lu MHz\n"
                     "  WiFi: %s (%s)\n"
                     "  RSSI: %d dBm\n",
                     (unsigned long)(HAL::Platform::getMillis() / 1000),
                     (unsigned long)HAL::getFreeHeap(),
                     HAL::getChipModel().c_str(), HAL::getChipRevision(),
                     (unsigned long)HAL::getCpuFreqMHz(),
                     HAL::WiFiHAL::getSSID().c_str(), HAL::WiFiHAL::getLocalIP().c_str(),
                     HAL::WiFiHAL::getRSSI());
            return String(buf);
        });
        
        // Heap command
        registerCommand("heap", [this](const String& args) {
            char buf[128];
            snprintf(buf, sizeof(buf), "Free Heap: %lu bytes\n", (unsigned long)HAL::getFreeHeap());
            return String(buf);
        });
        
        // Reboot command
        registerCommand("reboot", [this](const String& args) {
            for (auto& entry : clients) {
                entry.second.println("Rebooting...");
            }
            rebootRequestedAt = HAL::Platform::getMillis();
            rebootPending = true;
            return "";
        });
        
        // Quit command
        registerCommand("quit", [this](const String& args) {
            return "QUIT";  // Special return value to disconnect
        });
    }
    
    bool isIPAllowed(HAL::IPAddress ip) {
        if (config.allowedIPs.empty()) return true;
        
        for (const auto& allowed : config.allowedIPs) {
            if (ip == allowed) return true;
        }
        
        return false;
    }
    
    /** SEC-4 / SEC-14: one auth attempt, evaluated now. Failure sets the wait before the next one. */
    void evaluateAuth(uint32_t clientId, HAL::WiFiClient& client, const String& args) {
        ClientState& st = clientState[clientId];
        const unsigned long now = HAL::Platform::getMillis();
        const uint32_t ip = (uint32_t)client.remoteIP();
        if (!args.isEmpty() && !config.password.isEmpty() && config.password == args) {
            st.authenticated = true;
            authDelay.forget(ip);
            client.println("Authentication successful!");
            return;
        }
        const uint8_t failures = authDelay.noteFailure(ip, now);
        const unsigned long wait = Utils::AuthDelay::waitMs(failures, config.authDelayMaxMs);
        st.authNotBefore = now + wait;
        DLOG_W(LOG_CONSOLE, "Auth failure #%u from client #%u; next attempt read in %lu ms", failures, clientId, wait);
        client.println("Authentication failed.");
    }

    void sendWelcome(HAL::WiFiClient& client) {
        client.println("\n========================================");
        client.println("  DomoticsCore Remote Console");
        client.println("========================================");

        if (config.requireAuth) {
            client.println("Authentication required. Use: auth <password>\n");
        } else {
            client.println("Type 'help' for available commands\n");

            // Show recent logs only if not requiring auth
            auto recent = getRecentLogs(10);
            if (!recent.empty()) {
                client.println("Recent logs:");
                for (const auto& entry : recent) {
                    client.print(formatLogEntry(entry));
                }
                client.println();
            }
        }

        client.print("> ");  // Show initial prompt
    }
    
    void handleClient(uint32_t clientId, HAL::WiFiClient& client) {
        {
            // SEC-4: an auth attempt made too soon after a failure waits here, and
            // nothing else is read from this client until it has been evaluated.
            ClientState& st = clientState[clientId];
            if (st.authPending) {
                if ((long)(HAL::Platform::getMillis() - st.authNotBefore) < 0) return;
                st.authPending = false;
                String args = st.authPendingArgs;
                st.authPendingArgs = "";
                evaluateAuth(clientId, client, args);
                client.print("> ");
            }
        }
        // BUG-22, first half: bounded read, so a client streaming bytes without
        // pause cannot keep this loop from returning. At most MAX_BYTES_PER_LOOP
        // bytes per loop() call; the rest waits for the next iteration.
        static constexpr size_t MAX_BYTES_PER_LOOP = 512;

        // BUG-22, second half: bounding the read alone does not bound memory.
        // A client sending printable bytes and never a newline accumulates them
        // in commandBuffer across loop() calls — 512 bytes per call, with nothing
        // to stop it. On an ESP8266 that is remote heap exhaustion through an
        // open telnet port. The buffer is capped here, and the cap is the same
        // constant the line cleaner already truncates to below, so a long command
        // is cut at exactly the point it was being cut before — no new behaviour,
        // just a bound where there was none.
        static constexpr size_t MAX_COMMAND_LEN = 128;
        size_t bytesRead = 0;
        while (client.available() && bytesRead < MAX_BYTES_PER_LOOP) {
            char c = client.read();
            bytesRead++;

            // Get or create command buffer for this client
            String& commandBuffer = clientState[clientId].buffer;

            // Handle newline (command complete)
            if (c == '\n' || c == '\r') {
                // Skip if buffer is empty (just a newline from log output)
                if (commandBuffer.isEmpty()) {
                    continue;
                }

                String line = commandBuffer;
                commandBuffer = "";  // Clear buffer

                line.trim();

                // Remove any non-printable characters (telnet negotiation)
                // Constitution XIV: fixed-size char[] rather than String concat in a loop.
                // F5: 128 bytes rather than MAX_BYTES_PER_LOOP+1 (513), which was
                // more stack than an ESP8266 (~80KB RAM) should spend here.
                char cleaned[MAX_COMMAND_LEN];
                size_t ci = 0;
                for (size_t i = 0; i < line.length() && ci < MAX_COMMAND_LEN - 1; i++) {
                    char ch = line.charAt(i);
                    if (ch >= 32 && ch < 127) {  // Only printable ASCII
                        cleaned[ci++] = ch;
                    }
                }
                cleaned[ci] = '\0';
                line = String(cleaned);

                if (line.isEmpty()) continue;

                // Debug: log what we received
                DLOG_D(LOG_CONSOLE, "Command received: '%s' (len=%d)", line.c_str(), line.length());

                // Skip if command contains non-alphanumeric at start (telnet noise)
                if (line.length() > 0 && !isalnum(line.charAt(0)) && line.charAt(0) != ' ') {
                    DLOG_D(LOG_CONSOLE, "Skipping command with non-alphanumeric start: 0x%02X", line.charAt(0));
                    continue;
                }

                // Parse command and args
                int spacePos = line.indexOf(' ');
                String cmd = spacePos > 0 ? line.substring(0, spacePos) : line;
                String args = spacePos > 0 ? line.substring(spacePos + 1) : "";

                cmd.trim();
                args.trim();
                cmd.toLowerCase();

                // Handle auth command specially (needs client context)
                if (cmd == "auth") {
                    ClientState& st = clientState[clientId];
                    if (!config.requireAuth) {
                        client.println("Authentication not required.");
                    } else if ((long)(HAL::Platform::getMillis() - st.authNotBefore) < 0) {
                        st.authPending = true;   // SEC-4: held, evaluated when the wait has passed
                        st.authPendingArgs = args;
                        return;
                    } else {
                        evaluateAuth(clientId, client, args);
                    }
                    client.print("> ");
                    continue;
                }

                // Block commands if not authenticated (allow help + quit for discoverability)
                if (config.requireAuth
                    && !clientState[clientId].authenticated
                    && cmd != "help" && cmd != "quit") {
                    client.println("Authentication required. Use: auth <password>");
                    client.print("> ");
                    continue;
                }

                // Block commands if allowCommands is false (except help, quit)
                if (!config.allowCommands && cmd != "help" && cmd != "quit") {
                    client.println("Commands are disabled.");
                    client.print("> ");
                    continue;
                }

                // Execute command
                auto it = commands.find(cmd);
                if (it != commands.end()) {
                    String result = it->second(args);

                    if (result == "QUIT") {
                        client.println("Goodbye!");
                        client.stop();
                        return;
                    }

                    if (!result.isEmpty()) {
                        client.print(result);
                    }
                    client.print("> ");  // Show prompt after command
                } else {
                    char buf[128];
                    snprintf(buf, sizeof(buf), "Unknown command: %s (type 'help' for commands)", cmd.c_str());
                    client.println(buf);
                    client.print("> ");  // Show prompt after error
                }
            }
            // Handle backspace
            else if (c == '\b' || c == 127) {
                if (commandBuffer.length() > 0) {
                    commandBuffer.remove(commandBuffer.length() - 1);
                }
            }
            // Handle printable characters ONLY
            // F4: reserve once, up front, so a buffer partially filled by an
            // earlier loop() call does not reallocate on every appended byte.
            else if (c >= 32 && c < 127) {
                // Bounded by MAX_COMMAND_LEN, not by the per-loop read: bytes
                // beyond the cap are dropped rather than accumulated, so a client
                // that never sends a newline cannot grow this buffer without end.
                if (commandBuffer.length() >= MAX_COMMAND_LEN - 1) continue;
                commandBuffer.reserve(MAX_COMMAND_LEN);
                commandBuffer += c;
            }
            // Silently ignore all other control characters (telnet negotiation, etc.)
            // This includes: 0-31 (control chars), 127+ (extended ASCII, telnet IAC)
        }
    }
    
    static const char* getColorForLevel(LogLevel level) {
        switch (level) {
            case LOG_LEVEL_ERROR: return "\033[31m";
            case LOG_LEVEL_WARN:  return "\033[33m";
            case LOG_LEVEL_INFO:  return "\033[32m";
            case LOG_LEVEL_DEBUG: return "\033[36m";
            default:              return "";
        }
    }

    String formatLogEntry(const LogEntry& entry) {
        // Header: color(5) + timestamp(12) + level(6) + tag brackets/space(5) = ~28 fixed
        // Dynamic: tag + message — use stack buffer for typical case, fallback for long messages
        size_t needed = 40 + entry.tag.length() + entry.message.length();
        if (needed <= 256) {
            char buf[256];
            if (config.colorOutput) {
                snprintf(buf, sizeof(buf), "%s[%lu][%s][%s] %s\033[0m\n",
                         getColorForLevel(entry.level),
                         (unsigned long)entry.timestamp,
                         logLevelToString(entry.level).c_str(),
                         entry.tag.c_str(), entry.message.c_str());
            } else {
                snprintf(buf, sizeof(buf), "[%lu][%s][%s] %s\n",
                         (unsigned long)entry.timestamp,
                         logLevelToString(entry.level).c_str(),
                         entry.tag.c_str(), entry.message.c_str());
            }
            return String(buf);
        }
        // Fallback for long messages — single String concat
        String formatted;
        if (config.colorOutput) formatted += getColorForLevel(entry.level);
        char hdr[32];
        snprintf(hdr, sizeof(hdr), "[%lu][%s]", (unsigned long)entry.timestamp, logLevelToString(entry.level).c_str());
        formatted += hdr;
        formatted += "[";
        formatted += entry.tag;
        formatted += "] ";
        formatted += entry.message;
        if (config.colorOutput) formatted += "\033[0m";
        formatted += "\n";
        return formatted;
    }
    
    String logLevelToString(LogLevel level) {
        switch (level) {
            case LOG_LEVEL_NONE:  return "NONE";
            case LOG_LEVEL_ERROR: return "E";
            case LOG_LEVEL_WARN:  return "W";
            case LOG_LEVEL_INFO:  return "I";
            case LOG_LEVEL_DEBUG: return "D";
            default: return "?";
        }
    }
    
    void displayConnectionInfo() {
        if (connectionInfoDisplayed) return;
        
        if (HAL::WiFiHAL::isConnected()) {
            DLOG_I(LOG_CONSOLE, "Connect via: telnet %s %d", 
                   HAL::WiFiHAL::getLocalIP().c_str(), config.port);
            connectionInfoDisplayed = true;
        }
    }
};

} // namespace Components
} // namespace DomoticsCore
