#pragma once

/**
 * @file WebUI.h
 * @brief Declares the DomoticsCore WebUI component and supporting types for dashboard integration.
 */

#include "DomoticsCore/IWebUIProvider.h"
#include "DomoticsCore/ComponentRegistry.h"
#include "DomoticsCore/BaseWebUIComponents.h"
#include <ESPAsyncWebServer.h>
#include <ArduinoJson.h>
#include "DomoticsCore/Filesystem_HAL.h"
#include <vector>
#include <map>
#include <memory>
#include <algorithm>
#include <pgmspace.h>
#include <functional>

#include "DomoticsCore/IComponent.h"
#include "DomoticsCore/AuthDelay.h"
#include "DomoticsCore/Logger.h"
#include "DomoticsCore/Platform_HAL.h"  // For HAL::getFreeHeap()
#include "DomoticsCore/MemoryManager.h" // For adaptive WS limits
#include "DomoticsCore/WebUI_HAL.h"     // For WebUI buffer sizes
#include "DomoticsCore/Generated/WebUIAssets.h"

// New modular headers
#include "DomoticsCore/WebUI/WebUIConfig.h"
#include "DomoticsCore/WebUI/DeclaredApiHandler.h"
#include "DomoticsCore/WebUI/ProviderRegistry.h"
#include "DomoticsCore/WebUI/WebServerManager.h"
#include "DomoticsCore/WebUI/WebSocketHandler.h"
#include "DomoticsCore/WebUI/SystemHeader.h"
#include "DomoticsCore/WebUI/SchemaMemProbe.h"
#include "DomoticsCore/WebUI/UpdateBuilder.h"

namespace DomoticsCore {
namespace Components {

// Re-export WebUIConfig in the main namespace for backward compatibility
using WebUIConfig = WebUI::WebUIConfig;

/**
 * @class DomoticsCore::Components::WebUIComponent
 * @brief Async web server + WebSocket frontend that aggregates `IWebUIProvider` contexts.
 *
 * Serves embedded HTML/CSS/JS assets, registers component providers, and pushes real-time updates
 * to connected clients. Acts as both a component and a provider to expose global WebUI settings.
 */
class WebUIComponent : public IComponent, public CachingWebUIProvider, public Components::ComponentRegistry::IComponentLifecycleListener {
private:
    WebUIConfig config;
    
    // Sub-managers, destroyed in reverse: the registry outlives the server whose
    // handlers read it, and the server outlives webSocket, whose SSE source it owns.
    std::unique_ptr<WebUI::ProviderRegistry> registry;
    std::unique_ptr<WebUI::WebServerManager> webServer;
    std::unique_ptr<WebUI::WebSocketHandler> webSocket;

    // State
    bool forceNextUpdate = false; // force full contexts send on next tick (e.g., after WS reconnect)

    // SEC-10: per-boot CSRF token. 64 bits of hardware randomness, hex-encoded.
    // Regenerated every begin(); no persistence and no expiry by design — a
    // reader who can obtain it already has same-origin access, which is the one
    // case this cannot defend against anyway.
    char csrfToken_[17] = {0};

    // Failed-auth memory per address. Mutated only from the web server's task.
    mutable Utils::AuthDelay authDelay_;

    static uint32_t remoteAddress(AsyncWebServerRequest* request) {
        AsyncClient* client = request->client();
        return client ? static_cast<uint32_t>(client->remoteIP()) : 0;
    }

    // Update buffer shared by the poll handler (web task) and the SSE broadcast
    // (loop task): bufferLock_ covers it from build to last read, not only the build.
    // Taken before the registry's lock, never under it.
    static char wsBuffer_[WEBUI_WS_BUFFER_SIZE];
    static HAL::Platform::RecursiveLock bufferLock_;

    // Heap-staging diagnostics for /api/ui/schema (see WebUI/SchemaMemProbe.h)
    WebUI::SchemaMemProbes schemaProbes_;
    
    // Config persistence callback
    std::function<void(const WebUIConfig&)> onConfigChanged;

public:
    /**
     * @brief Construct a WebUI component with the provided configuration.
     */
    WebUIComponent(const WebUIConfig& cfg = WebUIConfig()) 
        : config(cfg) {
        // Initialize component metadata immediately for dependency resolution
        metadata.name = "WebUI";
        metadata.version = "1.13.0";  // x-release-please-version
        metadata.author = "DomoticsCore";
        metadata.description = "Web dashboard and API component";

        registry = std::unique_ptr<WebUI::ProviderRegistry>(new WebUI::ProviderRegistry());
        webServer = std::unique_ptr<WebUI::WebServerManager>(new WebUI::WebServerManager(config));
        webSocket = std::unique_ptr<WebUI::WebSocketHandler>(new WebUI::WebSocketHandler(config));
    }

    /**
     * @brief Release the instances.
     */
    ~WebUIComponent() {
        // unique_ptrs clean themselves up
    }

    // IComponent interface
    /**
     * @brief Initialize the AsyncWebServer, register websocket handler, and configure routes.
     */
    ComponentStatus begin() override {
        checkAuthConfig();
        // Adapt WS limits based on runtime memory profile
        auto& memMgr = MemoryManager::instance();
        int adaptiveMaxClients = memMgr.getMaxWsClients();
        if (adaptiveMaxClients < config.maxWebSocketClients) {
            DLOG_I(LOG_WEB, "Memory profile %s: WS clients %d -> %d",
                   memMgr.getProfileName(), config.maxWebSocketClients, adaptiveMaxClients);
            config.maxWebSocketClients = adaptiveMaxClients;
            // Recreate WebSocketHandler with updated config
            webSocket = std::unique_ptr<WebUI::WebSocketHandler>(new WebUI::WebSocketHandler(config));
        }

        generateCsrfToken();

        webServer->begin();
        webServer->setAuthHandler([this](AsyncWebServerRequest* request) {
            return authorize(request);
        });
        
        if (config.enableWebSocket) {
            // SEC-13: the stream honours enableAuth like every route, through
            // the live config rather than credentials copied at begin().
            webSocket->setAuthGate([this](AsyncWebServerRequest* request) { return authorize(request); });
            webSocket->begin(webServer->getServer());
            
            webSocket->setForceUpdateCallback([this]() {
                forceNextUpdate = true;
            });
            webSocket->setUIActionCallback([this](const String& ctx, const String& field, const String& value) {
                handleUIAction(ctx, field, value);
            });
        }
        
        setupApiRoutes();

        webServer->start();
        
        return ComponentStatus::Success;
    }

    /**
     * @brief Pump periodic websocket updates and clean up disconnected clients.
     */
    void loop() override {
        static unsigned long lastLoopLog = 0;
        unsigned long now = HAL::Platform::getMillis();
        if (now - lastLoopLog >= 10000) {
            DLOG_D(LOG_WEB, "WebUI loop alive, heap=%u, wsClients=%d, sse=%s(%d)",
                   (unsigned)HAL::Platform::getFreeHeap(), webSocket->getClientCount(),
                   webSocket->isSSEEnabled() ? "on" : "off", webSocket->getSSEClientCount());
            lastLoopLog = now;
        }

        webServer->poll();
        webSocket->loop();

        schemaProbes_.tick();

        registry->applyPendingEnables([this](const String& name) { webSocket->broadcastSchemaChange(name); });

        if (webSocket->shouldSendUpdates()) {
            sendWebSocketUpdates();
        }
    }

    /**
     * @brief Stop the web server but keep configuration for potential restart.
     */
    ComponentStatus shutdown() override {
        webServer->stop();
        return ComponentStatus::Success;
    }

    // Provider management facade
    void registerProvider(IWebUIProvider* provider) {
        registry->registerProvider(provider);
    }

    void registerProviderWithComponent(IWebUIProvider* provider, IComponent* component) {
        registry->registerProviderWithComponent(provider, component);
    }

    void unregisterProvider(IWebUIProvider* provider) {
        registry->unregisterProvider(provider);
    }

    int getWebSocketClients() const {
        return webSocket->getClientCount();
    }

    uint16_t getPort() const { 
        return config.port; 
    }

    void notifyWiFiNetworkChanged() {
        webSocket->notifyWiFiNetworkChanged();
    }
    
    void closeAllWebSocketConnections() {
        webSocket->closeAllConnections();
    }

    void setConfigCallback(std::function<void(const WebUIConfig&)> callback) {
        onConfigChanged = callback;
    }

    const WebUIConfig& getConfig() const {
        return config;
    }
    
    void setConfig(const WebUIConfig& cfg) {
        config = cfg;
        checkAuthConfig();
        DLOG_I(LOG_WEB, "Config updated: theme=%s, deviceName=%s", 
               config.theme, config.deviceName);
    }

    void registerProviderFactory(const String& typeKey, std::function<IWebUIProvider*(IComponent*)> factory) {
        registry->registerProviderFactory(typeKey, factory);
    }

    void registerApiRoute(const String& uri, WebRequestMethod method, ArRequestHandlerFunction handler) {
        webServer->registerRoute(uri, method, handler);
    }

    void registerApiUploadRoute(const String& uri, ArRequestHandlerFunction handler, ArUploadHandlerFunction uploadHandler) {
        webServer->registerUploadRoute(uri, handler, uploadHandler);
    }

    /**
     * @brief SEC-10: accept a request only if it carries this boot's CSRF token.
     *
     * The token is read from the `X-DC-Token` header or a `token` query
     * parameter — header for scripted clients, query for the browser, mirroring
     * how the OTA upload already accepts its hash either way. The check is
     * UNCONDITIONAL: it does not consult `enableAuth`, because auth-off is the
     * default and precisely the configuration the token must protect.
     * Authentication, where enabled, is an independent second layer applied by
     * each route after this. Public so sibling providers (OTAWebUI) gate their
     * own state-changing routes through the same check instead of copying it.
     */
    /**
     * @brief The auth gate every route shares: true when enableAuth is off or
     * the request carries this device's credentials, read live.
     * Public so sibling providers and the SSE middleware use it rather than a copy.
     * A wrong password makes its address wait (1 s, doubling, capped at
     * authDelayMaxMs); inside the wait a request is refused without reading it.
     */
    bool authorize(AsyncWebServerRequest* request) const {
        if (!config.enableAuth) return true;
        const uint32_t ip = remoteAddress(request);
        const unsigned long now = HAL::Platform::getMillis();
        if (authDelay_.isWaiting(ip, now, config.authDelayMaxMs)) return false;
        if (request->authenticate(config.username, config.password)) {
            authDelay_.forget(ip);
            return true;
        }
        // No Authorization header is the browser asking for the challenge, not a guess.
        if (request->hasHeader("Authorization")) {
            const uint8_t failures = authDelay_.noteFailure(ip, now);
            DLOG_W(LOG_WEB, "Auth failure #%u; next attempt read in %lu ms", failures,
                   Utils::AuthDelay::waitMs(failures, config.authDelayMaxMs));
        }
        return false;
    }

    bool checkCsrf(AsyncWebServerRequest* request) const {
        if (csrfToken_[0] == '\0') return false;  // not yet generated — fail closed
        String tok;
        if (const AsyncWebHeader* h = request->getHeader("X-DC-Token")) {
            tok = h->value();
        } else if (request->hasParam("token")) {
            tok = request->getParam("token")->value();
        } else {
            return false;
        }
        return tok.length() == 16 && strcmp(tok.c_str(), csrfToken_) == 0;
    }

    // IComponent override: post-initialization hook
    void onComponentsReady(const Components::ComponentRegistry& registry) override {
        this->registry->discoverProviders(registry);
        // Subscribe to future add/remove events. The hook's signature is const, the
        // registry is not: Core owns it as a plain member, so the cast is well defined.
        auto& reg = const_cast<Components::ComponentRegistry&>(registry);
        reg.addListener(this);
        
        // Subscribe to WiFi AP events - close WebSocket connections when network changes
        // This prevents crashes from sending to clients connected via the old network
        // Note: Using string literal to avoid WebUI depending on Wifi module
        // Event topic matches WifiEvents::EVENT_AP_ENABLED from DomoticsCore-Wifi
        on<bool>("wifi/ap/enabled", [this](const bool& enabled) {
            if (!enabled) {
                DLOG_I(LOG_WEB, "AP disabled - closing WebSocket connections");
                webSocket->closeAllConnections();
            }
        });
    }

    // CachingWebUIProvider implementation for self-registration
    String getWebUIName() const override { return "WebUI"; }
    String getWebUIVersion() const override { return metadata.version; }
    IWebUIProvider* getWebUIProvider() override { return this; }

protected:
    void buildContexts(std::vector<WebUIContext>& contexts) override {
        // IMPORTANT: Schema must use STATIC string literals only!
        // Dynamic values cause memory corruption when cached.
        // Real-time data comes from getWebUIData(), not from the schema.
        
        contexts.push_back(WebUIContext::headerInfo("webui_uptime", "Uptime", "dc-info")
            .withField(WebUIField("uptime", "Uptime", WebUIFieldType::Display, "--", "", true))
            .withRealTime(1000)
            .withAPI("/api/webui/uptime"));
        
        // Settings context - use static literals, not config.* members
        contexts.push_back(WebUIContext::settings("webui_settings", "Web Interface")
            .withField(WebUIField("theme", "Theme", WebUIFieldType::Select, "auto", "dark,light,auto"))
            .withField(WebUIField("primary_color", "Primary Color", WebUIFieldType::Text, "#007acc"))
            .withField(WebUIField("enable_auth", "Enable Authentication", WebUIFieldType::Boolean, "false"))
            .withField(WebUIField("username", "Username", WebUIFieldType::Text, "admin"))
            .withField(WebUIField("password", "Password", WebUIFieldType::Password, ""))
        );
    }

public:
    String getWebUIData(const String& contextId) override {
        if (contextId == "webui_uptime") {
            JsonDocument doc;
            uint32_t seconds = HAL::Platform::getMillis() / 1000;
            uint32_t days = seconds / 86400;
            seconds %= 86400;
            uint32_t hours = seconds / 3600;
            seconds %= 3600;
            uint32_t minutes = seconds / 60;
            seconds %= 60;

            String uptimeStr;
            if (days > 0) uptimeStr += String(days) + "d ";
            if (hours > 0 || days > 0) uptimeStr += String(hours) + "h ";
            if (minutes > 0 || hours > 0 || days > 0) uptimeStr += String(minutes) + "m ";
            uptimeStr += String(seconds) + "s";

            doc["uptime"] = uptimeStr;
            String json;
            serializeJson(doc, json);
            return json;
        }

        if (contextId == "webui_settings") {
            JsonDocument doc;
            doc["theme"] = config.theme;
            doc["primary_color"] = config.primaryColor;
            doc["enable_auth"] = config.enableAuth ? "true" : "false";
            doc["username"] = config.username;
            doc["password"] = ""; // Never send password back
            String json;
            serializeJson(doc, json);
            return json;
        }
        return "{}";
    }

    // Registry listener events
    void onComponentAdded(IComponent* comp) override {
        if (!comp) return;
        IWebUIProvider* provider = comp->getWebUIProvider();
        if (provider) {
            registerProviderWithComponent(provider, comp);
        }
    }

    void onComponentRemoved(IComponent* comp) override {
        registry->handleComponentRemoved(comp);
    }

    String handleWebUIRequest(const String& contextId, const String& endpoint, const String& method, const std::map<String, String>& params) override {
        if (contextId == "webui_settings" && method == "POST") {
            auto fieldIt = params.find("field");
            auto valueIt = params.find("value");
            if (fieldIt != params.end() && valueIt != params.end()) {
                const char* error = nullptr;
                if (!config.applySetting(fieldIt->second, valueIt->second, error)) {
                    return String("{\"success\":false, \"error\":\"") + error + "\"}";
                }
                
                if (onConfigChanged) {
                    onConfigChanged(config);
                }
                return "{\"success\":true}";
            }
        }
        return "{\"success\":false, \"error\":\"Invalid request\"}";
    }

private:

    /**
     * @brief SEC-10: mint a fresh 64-bit CSRF token for this boot.
     *
     * Hardware RNG via HAL::Platform::getRandomBytes — never Arduino random(),
     * which WMath.cpp downgrades to a millis-seeded rand() the moment any sketch
     * calls randomSeed(). Hex-encoded to 16 characters.
     */
    void generateCsrfToken() {
        uint8_t raw[8];
        HAL::Platform::getRandomBytes(raw, sizeof(raw));
        static const char hex[] = "0123456789abcdef";
        for (size_t i = 0; i < sizeof(raw); ++i) {
            csrfToken_[i * 2]     = hex[raw[i] >> 4];
            csrfToken_[i * 2 + 1] = hex[raw[i] & 0x0F];
        }
        csrfToken_[16] = '\0';
    }

    /**
     * @brief BUG-34: the ONE place the chunked-schema retry decision exists.
     *
     * writeChunk() returns 0 with `finished` still false when it cannot make
     * progress — transiently when the serializer's check-state chain is exited
     * in a call that wrote nothing yet (ordinary chunk sizes, ordinary
     * content), or until the buffer grows when an atomic escape sequence does
     * not fit. Returning that 0 from a chunked-response callback ends the
     * response: truncated JSON. v1.5.0 (b961e43a) fixed exactly that on the
     * poll route and /api/ui/schema never received the fix — because the
     * policy lived in two copied lambdas. Both routes now call this helper,
     * so the pair cannot drift again.
     */
    static size_t writeSchemaChunkHttp(WebUI::ProviderRegistry::SchemaChunkState& state,
                                       uint8_t* buffer, size_t maxLen) {
        size_t written = state.writeChunk(buffer, maxLen);
        if (written == 0 && !state.finished) return RESPONSE_TRY_AGAIN;
        return written;
    }

    /** SEC-14 / SEC-6: the two auth rules a config must satisfy, said once per change. */
    void checkAuthConfig() {
        if (config.normalizeAuth()) DLOG_W(LOG_WEB, "enableAuth with an empty password: authentication disabled");
        if (config.enableAuth && config.enableCORS) DLOG_W(LOG_WEB, "CORS headers are not sent while authentication is enabled");
    }

    /**
     * @brief Add CORS headers to response if enabled in config.
     * SEC-6: never with authentication on — a browser would not honour `*`
     * with credentials anyway, and a protected device should not advertise it.
     */
    void addCorsHeaders(AsyncWebServerResponse* response) {
        if (config.enableCORS && !config.enableAuth) {
            response->addHeader("Access-Control-Allow-Origin", "*");
            response->addHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS");
            response->addHeader("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization");
        }
    }
    
    // Whether some context declares this path with withAPI() and no route serves a GET on it.
    bool servesDeclaredApi(const String& path) const {
        if (webServer->hasRoute(path, HTTP_GET)) return false;
        return registry->withContextProviders([&](const std::map<String, IWebUIProvider*>& providers) {
            for (const auto& entry : providers) {
                bool declared = false;
                entry.second->forEachContext([&](const WebUIContext& ctx) {
                    if (entry.first != ctx.getContextIdCStr()) return true;
                    declared = path == ctx.getApiEndpointCStr();
                    return false;
                });
                if (declared) return true;
            }
            return false;
        });
    }

    // The data of every context declaring the path, keyed by context id.
    void answerDeclaredApi(AsyncWebServerRequest* request) {
        if (config.enableAuth && !authorize(request)) return request->requestAuthentication();
        if (HAL::Platform::getFreeHeap() < 4096) {
            return request->send(503, "application/json", "{\"error\":\"low memory\"}");
        }
        const String& path = request->url();
        String body = "{";
        registry->withContextProviders([&](const std::map<String, IWebUIProvider*>& providers) {
            for (const auto& entry : providers) {
                bool declared = false;
                entry.second->forEachContext([&](const WebUIContext& ctx) {
                    if (entry.first != ctx.getContextIdCStr()) return true;
                    declared = path == ctx.getApiEndpointCStr();
                    return false;
                });
                if (!declared) continue;
                if (body.length() > 1) body += ',';
                body += '"';
                body += entry.first;
                body += "\":";
                String data = entry.second->getWebUIData(entry.first);
                body += data.length() ? data : String("null");
            }
        });
        if (body.length() == 1) {   // its provider went away since canHandle()
            return request->send(404, "application/json", "{\"error\":\"Not found\"}");
        }
        body += '}';
        AsyncWebServerResponse* response = request->beginResponse(200, "application/json", body);
        addCorsHeaders(response);
        request->send(response);
    }

    void setupApiRoutes() {
        // Polling endpoint for real-time updates AND schema delivery
        // Use ?schema=1 to get schema (avoids separate TCP connection that fails during TIME_WAIT)
        webServer->registerRoute("/api/ui/updates", HTTP_GET, [this](AsyncWebServerRequest* request) {
            if (config.enableAuth && !authorize(request)) {
                return request->requestAuthentication();
            }
            
            // Check if client is requesting schema
            if (request->hasParam("schema")) {
                DLOG_I(LOG_WEB, "Schema via poll endpoint (heap: %u)", (unsigned)HAL::Platform::getFreeHeap());
                std::shared_ptr<WebUI::ProviderRegistry::SchemaChunkState> state = registry->prepareSchemaGeneration();
                AsyncWebServerResponse* response = request->beginChunkedResponse(
                    "application/json",
                    [state](uint8_t* buffer, size_t maxLen, size_t index) -> size_t {
                        if (!state) return 0;
                        return writeSchemaChunkHttp(*state, buffer, maxLen);
                    });
                addCorsHeaders(response);
                response->addHeader("Connection", "close");
                request->send(response);
                return;
            }
            
            // Normal polling update
            webSocket->onPollRequest();
            AsyncWebServerResponse* response = nullptr;
            {
                HAL::Platform::LockGuard guard(bufferLock_);
                int len = buildUpdateJson(true, forceNextUpdate);
                if (len > 0) {
                    // Inject SSE hint into response so frontend can upgrade to SSE
                    if (webSocket->isSSEEnabled()) {
                        if (len > 2 && wsBuffer_[len-1] == '}' && wsBuffer_[len-2] == '}') {
                            int extra = snprintf(wsBuffer_ + len - 1, sizeof(wsBuffer_) - len + 1,
                                ",\"_sse\":\"/api/ui/events\"}");
                            if (extra > 0) len += extra - 1;
                        }
                    }
                    // The response copies the buffer, so the lock can go before it is sent.
                    response = request->beginResponse(200, "application/json", wsBuffer_);
                }
            }
            if (response) {
                addCorsHeaders(response);
                response->addHeader("Connection", "close");
                request->send(response);
            } else {
                request->send(503, "application/json", "{\"error\":\"buffer overflow\"}");
            }
        });

        // SEC-10: per-boot CSRF token. The page fetches this once and echoes it
        // on every state-changing request. Cross-origin script cannot read this
        // response body while enableCORS stays false — that is what makes it a
        // CSRF defence rather than a formality. This route MUST NOT call
        // addCorsHeaders (ten neighbours do; this one must not), and carries no
        // expiry: a reader with same-origin access is already past every defence.
        webServer->registerRoute("/api/ui/token", HTTP_GET, [this](AsyncWebServerRequest* request) {
            if (config.enableAuth && !authorize(request)) {
                return request->requestAuthentication();
            }
            char body[40];
            snprintf(body, sizeof(body), "{\"token\":\"%s\"}", csrfToken_);
            request->send(200, "application/json", body);
        });

        // POST endpoint for client→server UI actions. Parameters stay in the
        // query string on purpose: query params are parsed for every method, and
        // body accumulation engages only for a urlencoded body, so POST-with-
        // query costs the same heap as the old GET did. (The former comment here
        // claimed a POST body parser costs ~500B and chose GET to avoid it — that
        // was wrong as applied, and the GET is what allowed the cross-origin
        // <img> shape.) Gated by the per-boot CSRF token; enableAuth, where on,
        // is an independent second layer.
        webServer->registerRoute("/api/ui/action", HTTP_POST, [this](AsyncWebServerRequest* request) {
            if (!checkCsrf(request)) {
                return request->send(403, "application/json", "{\"error\":\"Bad or missing CSRF token\"}");
            }
            if (config.enableAuth && !authorize(request)) {
                return request->requestAuthentication();
            }

            String contextId, field, value;
            if (request->hasParam("contextId")) {
                contextId = request->getParam("contextId")->value();
            }
            if (request->hasParam("field")) {
                field = request->getParam("field")->value();
            }
            if (request->hasParam("value")) {
                value = request->getParam("value")->value();
            }
            
            if (contextId.length() > 0 && field.length() > 0) {
                String response = handleUIAction(contextId, field, value);
                request->send(200, "application/json", response);
            } else {
                request->send(400, "application/json", "{\"error\":\"Missing contextId or field\"}");
            }
        });

        // System info API - optimized for ESP8266: use snprintf instead of String concatenation
        webServer->registerRoute("/api/system/info", HTTP_GET, [this](AsyncWebServerRequest* request) {
            if (config.enableAuth && !authorize(request)) return request->requestAuthentication();   // SEC-13
            char sysInfo[128];
            snprintf(sysInfo, sizeof(sysInfo), "{\"uptime\":%u,\"heap\":%u,\"clients\":%d}",
                (unsigned)HAL::Platform::getMillis(), (unsigned)HAL::Platform::getFreeHeap(), getWebSocketClients());
            AsyncWebServerResponse* response = request->beginResponse(200, "application/json", sysInfo);
            addCorsHeaders(response);
            request->send(response);
        });
        
        // API components
        webServer->registerRoute("/api/components", HTTP_GET, [this](AsyncWebServerRequest* request) {
            if (config.enableAuth && !authorize(request)) {
                return request->requestAuthentication();
            }
            
            AsyncResponseStream *response = request->beginResponseStream("application/json");
            addCorsHeaders(response);
            JsonDocument doc;
            registry->getComponentsList(doc);
            serializeJson(doc, *response);
            request->send(response);
        });

        // API components enable
        webServer->registerRoute("/api/components/enable", HTTP_POST, [this](AsyncWebServerRequest* request) {
            if (!checkCsrf(request)) {
                return request->send(403, "application/json", "{\"error\":\"Bad or missing CSRF token\"}");
            }
            if (config.enableAuth && !authorize(request)) {
                return request->requestAuthentication();
            }

            AsyncResponseStream *response = request->beginResponseStream("application/json");
            addCorsHeaders(response);
            
            String name;
            bool enabled = true;
            if (request->hasParam("name", true)) {
                name = request->getParam("name", true)->value();
            }
            if (request->hasParam("enabled", true)) {
                String v = request->getParam("enabled", true)->value();
                enabled = (v == "true" || v == "1" || v == "on");
            }

            // A component's lifecycle and the registry belong to the loop's task:
            // the request is queued here and applied by loop().
            JsonDocument doc;
            doc["name"] = name;
            doc["enabled"] = enabled;
            if (name == "WebUI" && !enabled) {
                doc["success"] = false;
                doc["warning"] = "Disabling WebUI may make the UI inaccessible until reboot/reset.";
            } else {
                doc["success"] = registry->requestEnable(name, enabled);
            }
            serializeJson(doc, *response);
            request->send(response);
        });

        // Context schema endpoint - loads full schema for a specific context
        webServer->registerRoute("/api/ui/context", HTTP_GET, [this](AsyncWebServerRequest* request) {
            if (config.enableAuth && !authorize(request)) {
                return request->requestAuthentication();
            }

            if (!request->hasParam("id")) {
                AsyncWebServerResponse* response = request->beginResponse(400, "application/json",
                    "{\"error\":\"Missing 'id' parameter\"}");
                addCorsHeaders(response);
                request->send(response);
                return;
            }

            String contextId = request->getParam("id")->value();
            DLOG_I(LOG_WEB, "Loading context schema for: %s (heap: %u)", contextId.c_str(), HAL::Platform::getFreeHeap());

            IWebUIProvider* provider = registry->getProviderForContext(contextId);
            if (!provider) {
                AsyncWebServerResponse* response = request->beginResponse(404, "application/json",
                    "{\"error\":\"Context not found\"}");
                addCorsHeaders(response);
                request->send(response);
                return;
            }

            // Find the context from the provider
            WebUIContext foundContext;
            bool found = false;
            provider->forEachContext([&](const WebUIContext& ctx) {
                if (strcmp(ctx.getContextIdCStr(), contextId.c_str()) == 0) {
                    foundContext = ctx;
                    found = true;
                    return false; // Stop iteration
                }
                return true; // Continue
            });

            if (!found) {
                AsyncWebServerResponse* response = request->beginResponse(404, "application/json",
                    "{\"error\":\"Context not found in provider\"}");
                addCorsHeaders(response);
                request->send(response);
                return;
            }

            // Serialize the context
            JsonDocument doc;
            JsonObject obj = doc.to<JsonObject>();
            serializeContext(obj, foundContext);

            AsyncResponseStream *response = request->beginResponseStream("application/json");
            addCorsHeaders(response);
            serializeJson(doc, *response);
            request->send(response);

            DLOG_I(LOG_WEB, "Context schema sent for: %s (heap: %u)", contextId.c_str(), HAL::Platform::getFreeHeap());
        });

        // Schema endpoint - uses ResponseStream for better memory management than chunked
        // ESPAsyncWebServer's chunked response has known memory leak issues
        webServer->registerChunkedRoute("/api/ui/schema", HTTP_GET, [this](AsyncWebServerRequest* request) {
            if (config.enableAuth && !authorize(request)) {
                request->requestAuthentication();
                return;
            }

            const WebUI::SchemaMemProbes::Armed probe = schemaProbes_.arm();

            request->onDisconnect([probe]() {
                DLOG_D(LOG_WEB, "Schema disconnect #%u: heap=%u (delta=%d), max=%u (delta=%d)",
                       (unsigned)probe.seq,
                       (unsigned)HAL::Platform::getFreeHeap(), (int)HAL::Platform::getFreeHeap() - (int)probe.heapBefore,
                       (unsigned)HAL::Platform::getMaxAllocHeap(), (int)HAL::Platform::getMaxAllocHeap() - (int)probe.maxBefore);
            });

            std::shared_ptr<WebUI::ProviderRegistry::SchemaChunkState> state = registry->prepareSchemaGeneration();

            AsyncWebServerResponse* response = request->beginChunkedResponse(
                "application/json",
                [state](uint8_t* buffer, size_t maxLen, size_t index) -> size_t {
                    if (!state) return 0;
                    // BUG-34: was `return state->writeChunk(...)` raw — a zero
                    // return ended the response mid-schema (see the helper).
                    return writeSchemaChunkHttp(*state, buffer, maxLen);
                });

            addCorsHeaders(response);
            response->addHeader("Connection", "close");
            request->send(response);

            schemaProbes_.logQueued(probe);
        });

        // Last, so every route above is found first; hasRoute() covers the later ones.
        if (webServer->getServer()) {
            webServer->getServer()->addHandler(new WebUI::DeclaredApiHandler(
                [this](const String& path) { return servesDeclaredApi(path); },
                [this](AsyncWebServerRequest* request) { answerDeclaredApi(request); }));
        }
    }
    
    // Duplicated helper to keep compilation working until I move it to a shared util or ProviderRegistry
    void serializeContext(JsonObject& obj, const WebUIContext& context) {
        obj["contextId"] = context.getContextIdCStr();
        obj["title"] = context.getTitleCStr();
        obj["icon"] = context.getIconCStr();
        obj["location"] = (int)context.location;
        obj["priority"] = context.priority;
        obj["apiEndpoint"] = context.getApiEndpointCStr();
        obj["alwaysInteractive"] = context.alwaysInteractive;
        
        if (context.hasCustomHtml()) obj["customHtml"] = context.getCustomHtmlCStr();
        if (context.hasCustomCss()) obj["customCss"] = context.getCustomCssCStr();
        if (context.hasCustomJs()) obj["customJs"] = context.getCustomJsCStr();

        JsonArray fields = obj["fields"].to<JsonArray>();
        for (const auto& field : context.fields) {
            JsonObject fieldObj = fields.add<JsonObject>();
            fieldObj["name"] = field.getNameCStr();
            fieldObj["label"] = field.getLabelCStr();
            fieldObj["type"] = (int)field.type;
            if (field.type == WebUIFieldType::Multiselect) {
                JsonArray values = fieldObj["value"].to<JsonArray>();
                for (const String& value : field.selectedValues) values.add(value);
            } else {
                fieldObj["value"] = field.getValueCStr();
            }
            fieldObj["unit"] = field.getUnitCStr();
            fieldObj["readOnly"] = field.readOnly;
            fieldObj["minValue"] = field.minValue;
            fieldObj["maxValue"] = field.maxValue;
            fieldObj["endpoint"] = field.getEndpointCStr();
            if (!field.options.empty()) {
                JsonArray options = fieldObj["options"].to<JsonArray>();
                for (const auto& opt : field.options) options.add(opt);
            }
            if (!field.optionLabels.empty()) {
                JsonObject labels = fieldObj["optionLabels"].to<JsonObject>();
                for (const auto& pair : field.optionLabels) labels[pair.first] = pair.second;
            }
        }
    }

    // In polling mode, initial state is served via GET /api/ui/updates
    
    // Handle UI action — returns provider's JSON response (may contain errors)
    String handleUIAction(const String& contextId, const String& field, const String& value) {
        IWebUIProvider* provider = registry->getProviderForContext(contextId);
        if (provider) {
            std::map<String, String> params;
            params["field"] = field;
            params["value"] = value;
            String result = provider->handleWebUIRequest(contextId, "/", "POST", params);
            forceNextUpdate = true;
            return result;
        }
        return "{\"success\":false,\"error\":\"Unknown context\"}";
    }

    // Build JSON update into wsBuffer_. Returns length, or 0 on failure.
    // forceFull=true sends all contexts (for polling), false uses delta check
    // (for SSE broadcast). The assembly lives in WebUI/UpdateBuilder.h (SIZE-1)
    // where native tests reach it — BUG-32's escaping and the crowding it can
    // cause are pinned there, not here.
    int buildUpdateJson(bool forceFull, bool forceNext) {
        return registry->withContextProviders([&](const std::map<String, IWebUIProvider*>& providers) {
            return WebUI::buildUpdateJson(wsBuffer_, sizeof(wsBuffer_), providers, config.deviceName,
                HAL::Platform::getMillis(), HAL::Platform::getFreeHeap(),
                getWebSocketClients(), forceFull, forceNext);
        });
    }

    void sendWebSocketUpdates() {
        // Skip entirely when no clients — saves building the JSON buffer
        if (webSocket->getClientCount() == 0) {
            forceNextUpdate = false;
            return;
        }

        // Skip when heap is critically low
        if (HAL::Platform::getFreeHeap() < 2000) {
            DLOG_W(LOG_WEB, "SSE broadcast skipped: heap=%u", (unsigned)HAL::Platform::getFreeHeap());
            return;
        }

        HAL::Platform::LockGuard guard(bufferLock_);
        // Cleared before the build: a client that connects during it keeps its full update.
        const bool force = forceNextUpdate;
        forceNextUpdate = false;
        int len = buildUpdateJson(false, force);
        if (len > 0) {
            DLOG_D(LOG_WEB, "SSE broadcast: %d bytes, clients=%d", len, webSocket->getClientCount());
            webSocket->broadcast(wsBuffer_, len);
        } else if (force) {
            forceNextUpdate = true;
        }
    }
};

// Static member definition (single shared WS buffer — saves 2KB BSS vs two separate buffers)
char WebUIComponent::wsBuffer_[WEBUI_WS_BUFFER_SIZE];
HAL::Platform::RecursiveLock WebUIComponent::bufferLock_;

} // namespace Components
} // namespace DomoticsCore
