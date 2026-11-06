





































































































    auto* ntpPtr = ntp.get();
    
    // Register sync callback
    ntpPtr->onSync([ntpPtr](bool success) {
        if (success) {
            DLOG_I(LOG_APP, "✅ Time synchronized!");
            DLOG_I(LOG_APP, "Current time: %s", ntpPtr->getFormattedTime().c_str());
























void loop() {
    core.loop();
    
    // Display time every 10 seconds
    if (displayTimer.isReady()) {
        
        auto* ntp = core.getComponent<NTPComponent>("NTP");
        if (ntp && ntp->isSynced()) {
            // Display current time in different formats
            DLOG_I(LOG_APP, "--- Current Time ---");
            DLOG_I(LOG_APP, "Full: %s", ntp->getFormattedTime("%Y-%m-%d %H:%M:%S").c_str());
            DLOG_I(LOG_APP, "Date: %s", ntp->getFormattedTime("%Y/%m/%d").c_str());
            DLOG_I(LOG_APP, "Time: %s", ntp->getFormattedTime("%H:%M:%S").c_str());
            DLOG_I(LOG_APP, "12h:  %s", ntp->getFormattedTime("%I:%M:%S %p").c_str());
            DLOG_I(LOG_APP, "Long: %s", ntp->getFormattedTime("%A, %B %d, %Y").c_str());
            DLOG_I(LOG_APP, "Unix: %lld", (long long)ntp->getUnixTime());
            DLOG_I(LOG_APP, "Uptime: %s", ntp->getFormattedUptime().c_str());
            
            // Next sync
            uint32_t nextSync = ntp->getNextSyncIn();
            if (nextSync > 0) {
                DLOG_I(LOG_APP, "Next sync in: %dm %ds", nextSync / 60, nextSync % 60);
            }
            
            // Statistics
            const auto& stats = ntp->getStatistics();
            DLOG_I(LOG_APP, "Stats: %d syncs, %d errors", stats.syncCount, stats.syncErrors);
            DLOG_I(LOG_APP, "-------------------");
        } else {
            DLOG_I(LOG_APP, "Time not synced yet...");
        }
    }
}
