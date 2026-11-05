#pragma once

/**
 * @file OTAWebUI.h
 * @brief Composition-based WebUI provider exposing OTA controls and REST endpoints.
 * 
 * @note OTA requires custom REST routes for file uploads (multipart/form-data).
 *       The `init()` method must be called after WebUI server initialization to register these routes.
 *       This differs from simpler components (LED, WiFi) which use only standard WebUI field interactions.
 */

#include <ArduinoJson.h>

#include "DomoticsCore/OTA.h"
#include "DomoticsCore/IWebUIProvider.h"
#include "DomoticsCore/WebUI.h"
#include "DomoticsCore/BaseWebUIComponents.h"
#include "DomoticsCore/Logger.h"

namespace DomoticsCore {
namespace Components {
namespace WebUI {

/**
 * @brief WebUI provider that bridges `OTAComponent` with `WebUIComponent` routes and contexts.
 */
class OTAWebUI : public CachingWebUIProvider {
public:
    explicit OTAWebUI(OTAComponent* component)
        : ota(component) {}

    /** @brief Called with the new config after a settings edit, for persistence. */
    void setConfigSaveCallback(std::function<void(const OTAConfig&)> callback) {
        onConfigSaved = callback;
    }

    /**
     * @brief Initialize routes after construction. Call this once WebUI component is available.
     * @param webuiComponent Pointer to the WebUI component for registering custom upload routes.
     */
    void init(WebUIComponent* webuiComponent) {
        webui = webuiComponent;
        if (webui) {
            registerRoutes();
        }
    }

    String getWebUIName() const override { return ota ? ota->metadata.name : String("OTA"); }
    String getWebUIVersion() const override { return ota ? ota->metadata.version : String("1.4.0"); }

protected:
    void buildContexts(std::vector<WebUIContext>& contexts) override {
        if (!ota) return;

        const OTAConfig& cfg = ota->getConfig();

        // Unified OTA card using standard components - placeholder values
        WebUIContext otaCard = WebUIContext::settings("ota_unified", "Firmware Update")
            .withAlwaysInteractive();

        if (cfg.enableWebUIUpload) {
            otaCard
                .withField(WebUIField("status", "Status", WebUIFieldType::Display, "Idle", "", true))
                .withField(WebUIField("progress", "Progress", WebUIFieldType::Progress, "0%", "", true))
                // Remote Update Section
                .withField(WebUIField("update_url", "Update URL", WebUIFieldType::Text, ""))
                .withField(WebUIField("check_now", "Check for Updates", WebUIFieldType::Button, ""))
                .withField(WebUIField("start_update", "Download & Install", WebUIFieldType::Button, ""))
                // Local Upload Section
                .withField(WebUIField("firmware", "Upload Firmware", WebUIFieldType::File, "", ".bin,.bin.gz").api("/api/ota/upload"))
                // Settings
                .withField(WebUIField("auto_reboot", "Auto Reboot", WebUIFieldType::Boolean, "true"));

            otaCard.withRealTime(2000).withAPI("/api/ota/unified");
        } else {
            // Remote-only mode (no upload)
            otaCard
                .withField(WebUIField("status", "Status", WebUIFieldType::Display, "Idle", "", true))
                .withField(WebUIField("progress", "Progress", WebUIFieldType::Display, "0%", "", true))
                .withField(WebUIField("downloaded", "Downloaded", WebUIFieldType::Display, "0", " bytes", true))
                .withField(WebUIField("update_url", "Firmware URL", WebUIFieldType::Text, ""))
                .withField(WebUIField("check_now", "Check For Updates", WebUIFieldType::Button, ""))
                .withField(WebUIField("start_update", "Download & Install", WebUIFieldType::Button, ""))
                .withField(WebUIField("auto_reboot", "Auto Reboot", WebUIFieldType::Boolean, "true"))
                .withRealTime(2000)
                .withAPI("/api/ota/update");
        }
        contexts.push_back(otaCard);
    }

public:

    String getWebUIData(const String& contextId) override {
        if (!ota) return "{}";

        JsonDocument doc;

        if (contextId == "ota_unified") {
            // Unified card data
            doc["state"] = stateToString(ota->getState());
            doc["message"] = ota->getLastResult();
            doc["progress"] = ota->getProgress(); // Send numeric value
            doc["bytes"] = ota->getDownloadedBytes();
            doc["total"] = ota->getTotalBytes();
            doc["update_url"] = ota->getConfig().updateUrl;
            doc["auto_reboot"] = ota->getConfig().autoReboot;
            doc["buttonEnabled"] = !ota->isBusy();
        } else {
            return "{}";
        }

        String json;
        if (serializeJson(doc, json) == 0) {
            return "{}";
        }
        return json;
    }

    bool hasDataChanged(const String& contextId) override {
        if (!ota) return false;
        
        if (contextId == "ota_unified") {
            // Only send updates when OTA state or progress changes
            OTAState current = {
                ota->getState(),
                ota->getProgress(),
                ota->getDownloadedBytes()
            };
            return otaState.hasChanged(current);
        }
        
        return true;  // Other contexts: always send
    }

    String handleWebUIRequest(const String& contextId, const String& /*endpoint*/, const String& method,
                              const std::map<String, String>& params) override {
        if (!ota) {
            return "{\"success\":false,\"error\":\"Component not available\"}";
        }

        if (method == "GET") {
            // Real-time updates handled by getWebUIData()
            return "{\"success\":true}";
        }

        if (method != "POST") {
            return "{\"success\":false,\"error\":\"Method not allowed\"}";
        }

        auto fieldIt = params.find("field");
        auto valueIt = params.find("value");
        if (fieldIt == params.end()) {
            return "{\"success\":false,\"error\":\"Invalid request\"}";
        }
        const String field = fieldIt->second;
        const String value = (valueIt != params.end()) ? valueIt->second : String();

        if (contextId == "ota_unified" || contextId == "ota_manager") {
            if (field == "update_url") {
                // Use Get → Override → Set pattern
                OTAConfig cfg = ota->getConfig();
                cfg.updateUrl = value;
                ota->setConfig(cfg);
                if (onConfigSaved) onConfigSaved(cfg);
                return "{\"success\":true}";
            }
            if (field == "check_now") {
                ota->triggerImmediateCheck(true);
                return "{\"success\":true}";
            }
            if (field == "start_update") {
                String url = value.isEmpty() || value == "clicked" ? ota->getConfig().updateUrl : value;
                if (url.isEmpty()) {
                    return "{\"success\":false,\"error\":\"No firmware URL configured\"}";
                }
                if (!ota->triggerUpdateFromUrl(url, true)) {
                    return "{\"success\":false,\"error\":\"Firmware hash required\"}";
                }
                return "{\"success\":true}";
            }
            if (field == "auto_reboot") {
                bool enable = (value == "true" || value == "1" || value == "on");
                // Use Get → Override → Set pattern
                OTAConfig cfg = ota->getConfig();
                cfg.autoReboot = enable;
                ota->setConfig(cfg);
                if (onConfigSaved) onConfigSaved(cfg);
                return "{\"success\":true}";
            }
        }

        return "{\"success\":false,\"error\":\"Unknown field\"}";
    }

private:
    OTAComponent* ota = nullptr;      //!< Non-owning pointer to the OTA component
    std::function<void(const OTAConfig&)> onConfigSaved;
    WebUIComponent* webui = nullptr;  //!< Non-owning pointer to the WebUI component (set via init())

    struct UploadState {
        bool active = false;
        bool success = false;
        bool rejected = false;    ///< Upload refused at index 0 — skip every chunk that follows.
                                  ///< Auth failure (SEC-3) or a refused beginUpload (SEC-7).
        String error;
        String filename;
        size_t total = 0;
        const AsyncWebServerRequest* gatedRequest = nullptr;  ///< The request chunk 0 asked the gate for.
        bool gateAuthorized = false;                          ///< And its answer.
    } uploadState;
    
    // State tracking for change detection
    struct OTAState {
        OTAComponent::State state;
        float progress;
        size_t bytes;
        
        bool operator==(const OTAState& other) const {
            return state == other.state && 
                   progress == other.progress && 
                   bytes == other.bytes;
        }
        bool operator!=(const OTAState& other) const { return !(*this == other); }
    };
    LazyState<OTAState> otaState;

    static String stateToString(OTAComponent::State state) {
        switch (state) {
            case OTAComponent::State::Idle: return "idle";
            case OTAComponent::State::Checking: return "checking";
            case OTAComponent::State::Downloading: return "downloading";
            case OTAComponent::State::Applying: return "applying";
            case OTAComponent::State::RebootPending: return "reboot_pending";
            case OTAComponent::State::Error: return "error";
        }
        return "unknown";
    }

    String formatProgress() const {
        return String(ota ? ota->getProgress() : 0.0f, 1) + "%";
    }

    void registerRoutes() {
        if (!webui) return;

        // Unified API endpoint for OTA card (GET for current state, POST for updates)
        webui->registerApiRoute("/api/ota/unified", HTTP_GET, [this](AsyncWebServerRequest* request) {
            // The page draws its card from /api/ui/updates, not from here.
            if (!webui->authorize(request)) {
                request->requestAuthentication();
                return;
            }
            respondJson(request, [this](JsonDocument& doc) {
                if (!ota) return;
                doc["state"] = stateToString(ota->getState());
                doc["message"] = ota->getLastResult();
                doc["progress"] = ota->getProgress();
                doc["bytes"] = ota->getDownloadedBytes();
                doc["total"] = ota->getTotalBytes();
                doc["update_url"] = ota->getConfig().updateUrl;
                doc["auto_reboot"] = ota->getConfig().autoReboot;
            });
        });

        webui->registerApiRoute("/api/ota/unified", HTTP_POST, [this](AsyncWebServerRequest* request) {
            if (!webui->authorize(request)) {
                request->requestAuthentication();
                return;
            }
            if (!ota) {
                respondJson(request, [](JsonDocument& doc) {
                    doc["success"] = false;
                    doc["error"] = "OTA unavailable";
                });
                return;
            }

            // Return current state (auto_reboot is read-only from config)
            respondJson(request, [this](JsonDocument& doc) {
                doc["state"] = stateToString(ota->getState());
                doc["message"] = ota->getLastResult();
                doc["progress"] = ota->getProgress();
                doc["bytes"] = ota->getDownloadedBytes();
                doc["total"] = ota->getTotalBytes();
            });
        });

        webui->registerApiRoute("/api/ota/status", HTTP_GET, [this](AsyncWebServerRequest* request) {
            if (!webui->authorize(request)) {
                request->requestAuthentication();
                return;
            }
            respondJson(request, [this](JsonDocument& doc) {
                if (!ota) return;
                doc["state"] = stateToString(ota->getState());
                doc["progress"] = ota->getProgress();
                doc["downloaded"] = ota->getDownloadedBytes();
                doc["total"] = ota->getTotalBytes();
                doc["lastResult"] = ota->getLastResult();
                doc["lastVersion"] = ota->getLastVersion();
                doc["autoReboot"] = ota->getConfig().autoReboot;
            });
        });

        webui->registerApiRoute("/api/ota/check", HTTP_POST, [this](AsyncWebServerRequest* request) {
            if (!webui->authorize(request)) {
                request->requestAuthentication();
                return;
            }
            if (!webui->checkCsrf(request)) {
                request->send(403, "application/json", "{\"success\":false,\"error\":\"Bad or missing CSRF token\"}");
                return;
            }
            if (ota) ota->triggerImmediateCheck(true);
            respondJson(request, [](JsonDocument& doc) {
                doc["success"] = true;
            });
        });

        webui->registerApiRoute("/api/ota/update", HTTP_POST, [this](AsyncWebServerRequest* request) {
            // Above the branch: the read below answers with the configured URL.
            if (!webui->authorize(request)) {
                request->requestAuthentication();
                return;
            }
            if (!ota) {
                respondJson(request, [](JsonDocument& doc){
                    doc["success"] = false;
                    doc["error"] = "OTA unavailable";
                });
                return;
            }
            
            // Check if this is a real-time update request (no parameters)
            bool hasParams = request->hasParam("url", true) || request->hasParam("force", true) || request->hasParam("action", true);
            
            if (!hasParams) {
                // Real-time update: return current field values
                respondJson(request, [this](JsonDocument& doc) {
                    doc["status"] = ota->getLastResult();
                    doc["progress"] = formatProgress();
                    doc["downloaded"] = String(ota->getDownloadedBytes());
                    doc["update_url"] = ota->getConfig().updateUrl;
                    doc["auto_reboot"] = ota->getConfig().autoReboot ? "true" : "false";
                });
                return;
            }
            
            if (!webui || !webui->checkCsrf(request)) {
                request->send(403, "application/json", "{\"success\":false,\"error\":\"Bad or missing CSRF token\"}");
                return;
            }
            String url = request->hasParam("url", true) ? request->getParam("url", true)->value() : ota->getConfig().updateUrl;
            bool force = false;
            if (request->hasParam("force", true)) {
                String v = request->getParam("force", true)->value();
                force = (v == "true" || v == "1" || v == "on");
            }
            const String sha256 = request->hasParam("sha256", true) ? request->getParam("sha256", true)->value() : String("");
            bool ok = !url.isEmpty() && ota->triggerUpdateFromUrl(url, sha256, force);
            respondJson(request, [ok, &url, this](JsonDocument& doc) {
                doc["success"] = ok;
                if (!ok) doc["error"] = url.isEmpty() ? "Missing or invalid URL" : ota->getLastError().c_str();
            });
        });

        if (ota && ota->getConfig().enableWebUIUpload) {
            // Simple HTML upload page for demonstration
            static const char OTA_UPLOAD_HTML[] PROGMEM =
                    "<!DOCTYPE html><html><head><meta charset='utf-8'><title>OTA Upload</title>"
                    "<meta name='viewport' content='width=device-width, initial-scale=1'>"
                    "<style>body{font-family:sans-serif;margin:2rem;} .card{max-width:480px;padding:1rem;border:1px solid #ccc;border-radius:8px;} button{padding:.5rem 1rem;} input{margin:.5rem 0;}</style>"
                    "</head><body><div class='card'>"
                    "<h2>Firmware Upload</h2>"
                    "<form method='POST' action='/api/ota/upload' enctype='multipart/form-data'>"
                    "<input type='file' name='firmware' accept='.bin,.bin.gz' required><br>"
                    "<button type='submit'>Upload & Install</button>"
                    "</form>"
                    "<p>After a successful upload, the device may reboot automatically.</p>"
                    "</div></body></html>";
            // SEC-3: Gate upload HTML page behind WebUI auth
            webui->registerApiRoute("/ota/upload", HTTP_GET, [this](AsyncWebServerRequest* request){
                if (!webui->authorize(request)) {
                    request->requestAuthentication();
                    return;
                }
                request->send(200, "text/html", OTA_UPLOAD_HTML);
            });
            webui->registerApiUploadRoute(
                "/api/ota/upload",
                [this](AsyncWebServerRequest* request) {
                    // SEC-10: this is the measured CRITICAL — an unauthenticated
                    // cross-origin multipart POST installs firmware and reboots.
                    // Gate on the WebUI's per-boot CSRF token, which cross-origin
                    // script cannot read. The token is also checked at upload
                    // chunk index 0 (below), before any flash is erased; this
                    // completion gate is what turns a refusal into a clean 403.
                    if (!webui || !webui->checkCsrf(request)) {
                        request->send(403, "application/json", "{\"success\":false,\"error\":\"Bad or missing CSRF token\"}");
                        return;
                    }
                    // Chunk 0 already asked the gate; asking again would count one wrong
                    // password twice, or refuse a good upload inside a wait set meanwhile.
                    const bool authorized = uploadState.gatedRequest == request ? uploadState.gateAuthorized
                                                                                : webui->authorize(request);
                    uploadState.gatedRequest = nullptr;
                    if (!authorized) {
                        request->requestAuthentication();
                        return;
                    }
                    respondJson(request, [this](JsonDocument& doc) {
                        doc["success"] = uploadState.success;
                        if (!uploadState.success) {
                            doc["error"] = uploadState.error;
                        } else {
                            doc["message"] = "Upload successful";
                        }
                    });
                },
                [this](AsyncWebServerRequest* request, const String& filename, uint32_t index, uint8_t* data, size_t len, bool final) {
                    // SEC-3: Reset state FIRST at index == 0, THEN check auth.
                    // This prevents a stale rejected flag from a previous failed upload
                    // from causing the current upload to be silently rejected.
                    if (index == 0) {
                        uploadState = UploadState{};  // Reset ALL state (clears stale rejected)
                        // BUG-35: a client that vanishes mid-body leaves the update
                        // open forever — beginUpload() then refuses every retry with
                        // "already in progress" until a power-cycle, on a device OTA
                        // exists to spare the trip to. onDisconnect fires after EVERY
                        // request, successful ones included, so the abort is gated on
                        // uploadState.active: false until the gates below pass, false
                        // again once `final` ran. Invariant that keeps the refusal
                        // paths safe: active is never set before every refusal path
                        // has returned. (Known limit, pre-existing: uploadState is
                        // one shared field, so a second concurrent uploader's reset
                        // can blind this predicate — single upload session at a time
                        // is the standing assumption of this handler.)
                        request->onDisconnect([this]() {
                            if (!uploadState.active) return;
                            uploadState.active = false;
                            uploadState.success = false;
                            uploadState.error = "Client disconnected mid-upload";
                            ota->abortUpload("Client disconnected mid-upload");
                        });
                        // SEC-10: refuse before beginUpload() erases flash. The
                        // completion handler returns the 403; here the point is
                        // to reject at index 0 so no flash write is ever opened
                        // for a request that lacks this boot's CSRF token.
                        if (!webui || !webui->checkCsrf(request)) {
                            uploadState.success = false;
                            uploadState.error = "Bad or missing CSRF token";
                            uploadState.rejected = true;
                            ota->abortUpload("CSRF token missing");
                            return;
                        }
                        uploadState.gatedRequest = request;
                        uploadState.gateAuthorized = webui->authorize(request);
                        if (!uploadState.gateAuthorized) {
                            uploadState.success = false;
                            uploadState.error = "Authentication required";
                            uploadState.rejected = true;
                            // Abort any in-progress OTA to prevent flash writes
                            ota->abortUpload("Authentication required");
                            return;
                        }
                        uploadState.active = true;
                        // BUG-37: ESPAsyncWebServer arms a 3 s receive-idle timeout on
                        // every accepted client (WebServer.cpp, setRxTimeout(3)) and
                        // clears it only when a response starts, so it runs for the
                        // whole body of an upload — and a client in TCP retransmission
                        // backoff is quiet for longer than that on an ordinary WiFi
                        // link. The measurements are in BUG-37's roadmap entry.
                        // Widened for this request only, after the gates above: a
                        // request without this boot's CSRF token, or failing auth
                        // where auth is on, keeps the server's 3 s. The bound this
                        // buys an attacker is small — one upload at a time, the
                        // Update held for at most uploadIdleTimeoutSec per silent
                        // client, BUG-35's abort as the release. A client that
                        // beginUpload() refuses below keeps the wider limit while its
                        // body drains, accepted. The response path resets it to 0 as
                        // it always did. Not 0 by default: a client that vanishes
                        // without a RST would leave the update open, BUG-35's lock
                        // through another door.
                        if (AsyncClient* client = request->client()) {
                            client->setRxTimeout(ota->getConfig().uploadIdleTimeoutSec);
                        }
                        uploadState.filename = filename;
                        uploadState.total = 0;
                        // SEC-9: this measures the whole multipart body — boundary,
                        // part headers, trailing boundary — and not the firmware.
                        // 220 bytes more on both boards. It is passed on anyway,
                        // deliberately: an upper bound never truncates Update.begin(),
                        // and it lets beginUpload() refuse a grossly oversized upload
                        // before an update is opened at all. Passing 0 instead removes
                        // that refusal from this path and widens the shortfall
                        // end(true) has to absorb. See SEC-9 in docs/CODE-ROADMAP.md
                        // before changing this line.
                        size_t expectedSize = request->contentLength();

                        // SEC-7: the expected digest, if the client sent one.
                        // Header first, query parameter as a fallback for tools
                        // that cannot set headers. Both are parsed before the
                        // body, so both are available here at index 0 — a
                        // multipart field would not be, since its position in
                        // the body decides when it arrives.
                        String expectedSha;
                        if (const AsyncWebHeader* h = request->getHeader("X-Firmware-SHA256")) {
                            expectedSha = h->value();
                        } else if (request->hasParam("sha256")) {
                            expectedSha = request->getParam("sha256")->value();
                        }

                        // beginUpload() refuses an upload with no hash when
                        // requireUploadHash is set, and refuses before erasing
                        // any flash. Discarding that result surfaced the refusal
                        // one chunk later as "Upload not active", which says
                        // nothing about why.
                        if (!ota->beginUpload(expectedSize, expectedSha)) {
                            uploadState.success = false;
                            uploadState.error = ota->getLastError();
                            uploadState.rejected = true;
                            return;
                        }
                    }
                    // Skip every chunk once the upload has been refused
                    if (uploadState.rejected) return;
                    if (data && len > 0) {
                        uploadState.total += len;
                        if (!ota->acceptUploadChunk(data, len)) {
                            uploadState.success = false;
                            uploadState.error = ota->getLastError();
                            uploadState.rejected = true;
                            // The component already closed its session; the
                            // disconnect handler must not abort it again or
                            // overwrite this error.
                            uploadState.active = false;
                        }
                    }
                    if (final) {
                        uploadState.active = false;
                        uploadState.success = ota->finalizeUpload();
                        if (!uploadState.success) {
                            uploadState.error = ota->getLastError();
                        }
                    }
                }
            );
        }
    }

    template<typename Fn>
    static void respondJson(AsyncWebServerRequest* request, Fn fn) {
        AsyncResponseStream* response = request->beginResponseStream("application/json");
        JsonDocument doc;
        fn(doc);
        serializeJson(doc, *response);
        request->send(response);
    }
};

} // namespace WebUI
} // namespace Components
} // namespace DomoticsCore
