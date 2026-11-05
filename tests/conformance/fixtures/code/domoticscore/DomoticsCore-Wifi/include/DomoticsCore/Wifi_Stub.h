#ifndef DOMOTICS_CORE_WIFI_STUB_H
#define DOMOTICS_CORE_WIFI_STUB_H

/**
 * @file Wifi_Stub.h
 * @brief Stub WiFi implementation for unsupported platforms.
 */

#if !DOMOTICS_PLATFORM_ESP32 && !DOMOTICS_PLATFORM_ESP8266

#include <vector>
#include <utility>

namespace DomoticsCore {
namespace HAL {
namespace WiFiImpl {

inline bool stubbedConnected = false;
inline void setConnectedForTest(bool connected) { stubbedConnected = connected; }

// ---------------------------------------------------------------------------
// Stateful mode / AP / connect record (TEST-4).
//
// This stub was stateless: getMode() always Off, startAP() always false,
// getAPSSID() always "". That made three of WifiComponent's paths unreachable
// or unobservable natively — the skip-restart branch compares getMode() and
// getAPSSID() against what startAP() was given, the STA fallback ladder is
// asserted through which HAL calls fired, and the reboot-to-STA path's only
// effect is a restart. Same pattern as the scan table above: scriptable,
// reset per test, and every DEFAULT is byte-identical to the old stub —
// startAP still returns false unless a test scripts it true, so suites that
// do not opt in see the exact behaviour this file had.
//
// The state is process-global. A suite that runs an AP fixture and does not
// call resetWifiStateForTest() in tearDown leaks AccessPoint mode into the
// next test's isAPMode().
// ---------------------------------------------------------------------------
struct StubWifiState {
    WiFiHAL::Mode mode = WiFiHAL::Mode::Off;
    bool apActive = false;
    String apSSID;
    bool lastAPPasswordWasNull = false;  // startAP is called with nullptr for an open AP
    bool startAPResult = false;          // the old stub's unconditional return
    unsigned int startAPCalls = 0;
    unsigned int stopAPCalls = 0;
    unsigned int connectCalls = 0;
    String lastConnectSSID;
    String lastConnectPassword;
    unsigned int disconnectCalls = 0;
    unsigned int disconnectAndOffCalls = 0;
    uint8_t apStationCount = 0;
    uint8_t rawStatus = 0;
};

inline StubWifiState& stubWifiState() {
    static StubWifiState state;
    return state;
}

inline void setStartAPResultForTest(bool result) { stubWifiState().startAPResult = result; }
inline void setAPStationCountForTest(uint8_t count) { stubWifiState().apStationCount = count; }
inline void setRawStatusForTest(uint8_t status) { stubWifiState().rawStatus = status; }
// Restores every default this file scripts — the mode/AP/connect record and
// the connected flag alike, so one tearDown call covers the whole stub.
inline void resetWifiStateForTest() {
    stubWifiState() = StubWifiState();
    stubbedConnected = false;
}

// ---------------------------------------------------------------------------
// Scriptable scan results.
//
// `scanNetworks()` returned a hard 0 here until 2026-08-29, and that meant the
// two scan-summary loops in Wifi.h — the ones MEM-2 rewrote — had **never
// executed on any platform CI can run**. Neither loop body is entered when the
// scan finds nothing, so the required native job would have stayed green
// through a rewrite that produced the wrong text, the wrong separator or a
// truncated entry. The device suite that measures them needs a radio and a
// neighbourhood; this needs neither, and it is what pins the format.
//
// The table is empty by default, so a suite that does not script it sees
// exactly the behaviour this file had before — zero networks, empty SSIDs.
// ---------------------------------------------------------------------------
struct StubNetwork {
    String ssid;
    int32_t rssi;
};

inline std::vector<StubNetwork>& stubbedNetworks() {
    static std::vector<StubNetwork> networks;
    return networks;
}

// 0 or above: the table's size is the result. Negative: returned as-is, which
// is how the failure branches are reached (-1 for the synchronous scan guard,
// -2 for WIFI_SCAN_FAILED on the asynchronous one).
inline int16_t stubbedScanResult = 0;
inline int scanDeleteCallsForTest = 0;  // counted by scanDelete() below

inline void setScannedNetworksForTest(std::vector<StubNetwork> networks) {
    stubbedNetworks() = std::move(networks);
    stubbedScanResult = 0;
}

inline void setScanFailedForTest(int16_t code) {
    stubbedNetworks().clear();
    stubbedScanResult = code;
}

inline void resetScanForTest() {
    stubbedNetworks().clear();
    stubbedNetworks().shrink_to_fit();
    stubbedScanResult = 0;
    scanDeleteCallsForTest = 0;
}

inline void init() {}
inline void setMode(WiFiHAL::Mode mode) { stubWifiState().mode = mode; }
inline void connect(const char* ssid, const char* password) {
    auto& s = stubWifiState();
    ++s.connectCalls;
    s.lastConnectSSID = ssid ? String(ssid) : String("");
    s.lastConnectPassword = password ? String(password) : String("");
}
inline void disconnect() { ++stubWifiState().disconnectCalls; }
inline bool startAP(const char* ssid, const char* password) {
    auto& s = stubWifiState();
    ++s.startAPCalls;
    // String(nullptr) is UB on the native String — record the null as a flag.
    s.apSSID = ssid ? String(ssid) : String("");
    s.lastAPPasswordWasNull = (password == nullptr);
    if (s.startAPResult) s.apActive = true;
    return s.startAPResult;
}
inline void stopAP() {
    auto& s = stubWifiState();
    ++s.stopAPCalls;
    s.apActive = false;
}
inline WiFiHAL::Status getStatus() { return WiFiHAL::Status::NotSupported; }
inline bool isConnected() { return stubbedConnected; }
inline String getLocalIP() { return "0.0.0.0"; }
inline String getAPIP() { return "0.0.0.0"; }
inline String getSSID() { return ""; }
inline int32_t getRSSI() { return 0; }
inline String getMacAddress() { return "00:00:00:00:00:00"; }
inline void setHostname(const char*) {}
inline void setAutoReconnect(bool) {}
inline int16_t scanNetworks(bool) {
    if (stubbedScanResult < 0) return stubbedScanResult;
    return static_cast<int16_t>(stubbedNetworks().size());
}
inline String getScannedSSID(uint8_t index) {
    const auto& networks = stubbedNetworks();
    return index < networks.size() ? networks[index].ssid : String("");
}
inline int32_t getScannedRSSI(uint8_t index) {
    const auto& networks = stubbedNetworks();
    return index < networks.size() ? networks[index].rssi : 0;
}
inline WiFiHAL::Mode getMode() { return stubWifiState().mode; }
// The recorded SSID only while the AP is up — the skip-restart branch in
// updateWifiMode() compares this against the component's target SSID.
inline String getAPSSID() { return stubWifiState().apActive ? stubWifiState().apSSID : String(""); }
inline uint8_t getAPStationCount() { return stubWifiState().apStationCount; }
inline int16_t scanComplete() {
    if (stubbedScanResult < 0) return stubbedScanResult;
    return static_cast<int16_t>(stubbedNetworks().size());
}
// Counted, not cleared: the scripted table is owned by whichever test set it.
// On a real SDK this frees the scan-result list.
inline void scanDelete() { ++scanDeleteCallsForTest; }
inline void disconnectAndOff() { ++stubWifiState().disconnectAndOffCalls; }
inline uint8_t getRawStatus() { return stubWifiState().rawStatus; }

} // namespace WiFiImpl

class NetworkClient {};
class SecureNetworkClient {};

} // namespace HAL
} // namespace DomoticsCore

#endif // Stub

#endif // DOMOTICS_CORE_WIFI_STUB_H
