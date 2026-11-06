

















































































































































































































void test_begin_logs_the_promoted_record_line_by_line() {
    FlightRecorder::instance().begin();
    HAL::Platform::setMillisForTest(1000);
    HAL::Platform::advanceMillisForTest(FlightRecorder::FAST_INTERVAL_MS);
    FlightRecorder::instance().tick();                                  // one ring sample
    CrashInfo death; death.reason = 254; death.failSize = 512;
    FlightRecorder::instance().recordCrash(death);
    FlightRecorder::instance().resetForTest();
    HAL::Platform::setResetReasonForTest(HAL::Platform::ResetReason::Software);
    g_lines.clear();
    auto id = LoggerCallbacks::addCallback([](LogLevel, const char*, const char* msg) { g_lines.emplace_back(msg ? msg : ""); });
    TEST_ASSERT_TRUE(testCore->begin());
    LoggerCallbacks::removeCallback(id);
    bool sawDeath = false, sawRing = false;
    for (const auto& l : g_lines) {
        if (l.find("Last death:") != std::string::npos) sawDeath = true;
        if (l.find("ring: 11s:") != std::string::npos) sawRing = true;
    }
    TEST_ASSERT_TRUE_MESSAGE(sawDeath, "the death line must reach the log");
    TEST_ASSERT_TRUE_MESSAGE(sawRing, "the ring must reach the log as its own line");
    HAL::Platform::resetMillisForTest();
}



































void test_a_failed_hook_registration_is_reported_not_claimed() {
    HAL::Platform::failedAllocHookInstallFailsForTest = true;
    g_lines.clear();
    auto id = LoggerCallbacks::addCallback([](LogLevel, const char*, const char* msg) { g_lines.emplace_back(msg ? msg : ""); });
    TEST_ASSERT_TRUE(testCore->begin());
    LoggerCallbacks::removeCallback(id);
    TEST_ASSERT_FALSE(FlightRecorder::instance().failedAllocHookInstalled());
    bool said = false;
    for (const auto& l : g_lines) if (l.find("Failed-allocation hook not registered") != std::string::npos) said = true;
    TEST_ASSERT_TRUE(said);
}







void test_loop_warns_about_survived_failures_at_most_once_a_minute() {
    HAL::Platform::setMillisForTest(1000);
    TEST_ASSERT_TRUE(testCore->begin());
    g_lines.clear();
    auto id = LoggerCallbacks::addCallback([](LogLevel, const char*, const char* msg) { g_lines.emplace_back(msg ? msg : ""); });
    HAL::Platform::advanceMillisForTest(60000);            // clear of any earlier test's minute
    HAL::Platform::fireFailedAllocForTest(512, 0x1800);
    testCore->loop();
    TEST_ASSERT_EQUAL_UINT(1, countAllocWarnings());
    HAL::Platform::fireFailedAllocForTest(512, 0x1800);
    HAL::Platform::advanceMillisForTest(1000);
    testCore->loop();                                       // same minute: silent
    TEST_ASSERT_EQUAL_UINT(1, countAllocWarnings());
    HAL::Platform::advanceMillisForTest(60000);
    testCore->loop();                                       // next minute: the new count
    TEST_ASSERT_EQUAL_UINT(2, countAllocWarnings());
    HAL::Platform::advanceMillisForTest(60000);
    testCore->loop();                                       // a minute with no new failure: silent (the change gate)
    TEST_ASSERT_EQUAL_UINT(2, countAllocWarnings());
    bool sawTwo = false;
    for (const auto& l : g_lines) if (l.find("since boot: 2, last 512 B") != std::string::npos) sawTwo = true;
    TEST_ASSERT_TRUE(sawTwo);
    LoggerCallbacks::removeCallback(id);
    HAL::Platform::resetMillisForTest();
}

void test_begin_logs_every_ring_line_of_a_saturated_record() {
    // The boot-log buffer must hold format()'s longest output, or the ring's
    // tail — the heap's trend into the death — is cut silently.
    FlightRecorder::instance().begin();
    FlightRecord r;
    HAL::Platform::rtcRead(0, r.w, FlightRecord::WORDS);
    for (size_t i = FlightRecord::W_SEQ; i < FlightRecord::WORDS; ++i) r.w[i] = 0xFFFFFFFFu;
    r.w[FlightRecord::W_PHASE] = FlightRecord::encodePhase(0xFFFF);
    r.w[FlightRecord::W_FAIL] = FlightRecord::encodeCount(0xFFFF);
    r.w[FlightRecord::W_META] |= FlightRecord::CALLBACK_RAN;
    r.w[FlightRecord::W_CRC] = r.bodyCrc();
    HAL::Platform::rtcWrite(0, r.w, FlightRecord::WORDS);
    FlightRecorder::instance().resetForTest();
    HAL::Platform::setResetReasonForTest(HAL::Platform::ResetReason::Software);
    g_lines.clear();
    auto id = LoggerCallbacks::addCallback([](LogLevel, const char*, const char* msg) { g_lines.emplace_back(msg ? msg : ""); });
    TEST_ASSERT_TRUE(testCore->begin());
    LoggerCallbacks::removeCallback(id);
    unsigned ringLines = 0;
    for (const auto& l : g_lines) if (l.find("ring:") != std::string::npos) ++ringLines;
    TEST_ASSERT_EQUAL_UINT(4, ringLines);                   // 16 samples, four per line; red with a 768-byte buffer
}



























