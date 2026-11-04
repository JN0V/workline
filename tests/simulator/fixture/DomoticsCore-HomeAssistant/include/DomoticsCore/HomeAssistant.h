// The Home Assistant component: discovery and availability over MQTT.
#pragma once

namespace DomoticsCore {

class HomeAssistantComponent {
public:
    void setAvailable(bool on) { available_ = on; ++published_; }
    void removeDiscovery() { ++published_; }
    void onMqttConnected(bool up) { mqttConnected = up; }

    // shutdown(): marks the device gone and removes its discovery.
    void shutdown() {
        setAvailable(false);
        removeDiscovery();
    }

    int published() const { return published_; }

private:
    bool mqttConnected = false;
    bool available_ = false;
    int published_ = 0;
};

}  // namespace DomoticsCore
