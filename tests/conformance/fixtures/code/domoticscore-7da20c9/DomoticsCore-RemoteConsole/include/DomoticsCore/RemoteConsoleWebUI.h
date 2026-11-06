
#define DOMOTICS_REMOTE_CONSOLE_WEBUI_H

#include <DomoticsCore/IWebUIProvider.h>
#include <DomoticsCore/RemoteConsole.h>
#include <DomoticsCore/WebUI.h>
#include <ArduinoJson.h>



namespace WebUI {

/**
 * @brief Minimal WebUI provider for RemoteConsole component
 *
 * Shows RemoteConsole in the component list. No configuration needed.
 */
class RemoteConsoleWebUI : public CachingWebUIProvider {
private:
    RemoteConsoleComponent* console;

    struct ConsoleUIState {
        bool active;












    explicit RemoteConsoleWebUI(RemoteConsoleComponent* c) : console(c) {}
























    String getWebUIName() const override {
        return console ? console->metadata.name : String("RemoteConsole");
    }










































































































