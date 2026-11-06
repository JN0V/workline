

#include "DomoticsCore/NTP.h"
#include "DomoticsCore/IWebUIProvider.h"
#include "DomoticsCore/WebUI.h"
#include <ArduinoJson.h>
#include <cstdint>

namespace DomoticsCore {
namespace Components {
namespace WebUI {

// ========== Constexpr Timezone Lookup (stored in flash, not RAM) ==========















































static constexpr size_t TIMEZONE_LOOKUP_COUNT = sizeof(TIMEZONE_LOOKUP) / sizeof(TIMEZONE_LOOKUP[0]);

/**
 * @brief WebUI provider for NTP component
 *
 * Provides web interface for NTP configuration, time display, and statistics.
 *






class NTPWebUI : public CachingWebUIProvider {
public:
    /**
     * @brief Construct WebUI provider
     * @param ntp Pointer to NTP component (non-owning)
     */
    explicit NTPWebUI(NTPComponent* ntp) : ntp(ntp) {
        // State tracking uses LazyState helper - no manual initialization needed








    }

    /**
     * @brief Initialize WebUI routes (call after WebUI component is ready)
     * Registers the /api/ntp/timezones endpoint for on-demand timezone loading
     */
    void init(WebUIComponent* webui) {












































































































    String handleWebUIRequest(const String& contextId, const String& /*endpoint*/,
                              const String& method, const std::map<String, String>& params) override {
        DLOG_D(LOG_NTP, "[WebUI] handleWebUIRequest: contextId=%s, method=%s", contextId.c_str(), method.c_str());
        
        if (!ntp) {
            DLOG_E(LOG_NTP, "[WebUI] NTP component not available");


















































































                // Trigger immediate sync after configuration change
                if (cfg.enabled) {
                    DLOG_I(LOG_NTP, "[WebUI] Triggering immediate sync after config save");
                    ntp->syncNow();
                }

                return "{\"success\":true}";










































