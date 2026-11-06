
















































    auto* ntpPtr = ntp.get();
    
    // Register sync callback
    ntpPtr->onSync([ntpPtr](bool success) {
        if (success) {
            Serial.printf("Time synced: %s\n", ntpPtr->getFormattedTime().c_str());
        } else {

































































    };
    uint32_t syncInterval = 3600;           // Sync interval in seconds
    String timezone = "UTC0";               // POSIX TZ string
    uint32_t timeoutMs = 5000;              // Connection timeout
};
```
























### Time Synchronization

```cpp
bool syncNow();                     // Trigger immediate sync
bool isSynced() const;              // Check if time is synced
time_t getLastSyncTime() const;     // Get last sync timestamp
uint32_t getNextSyncIn() const;     // Seconds until next sync



























### Callbacks

```cpp
void onSync(SyncCallback callback); // Register sync callback
```

### Statistics

```cpp
struct NTPStatistics {
    uint32_t syncCount;             // Total successful syncs
    uint32_t syncErrors;            // Total sync failures
    time_t lastSyncTime;            // Last successful sync
    uint32_t lastSyncDuration;      // Last sync duration (ms)
    time_t lastFailTime;            // Last failure time
    uint32_t consecutiveFailures;   // Consecutive failures
};

const NTPStatistics& getStatistics() const;
```

## Time Format Strings
































































**Check sync status:**
```cpp
auto* ntp = core.getComponent<NTPComponent>("NTP");
const auto& stats = ntp->getStatistics();
Serial.printf("Sync attempts: %d, Errors: %d\n", stats.syncCount, stats.syncErrors);
```

### Wrong timezone

















































