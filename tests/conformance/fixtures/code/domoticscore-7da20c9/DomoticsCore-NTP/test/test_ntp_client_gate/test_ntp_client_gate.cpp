


 *
 * On ESP32 lwIP exists from the first network interface on; starting the
 * client before that aborts the firmware. The component asks
 * HAL::canOpenServer() in begin() and starts from loop() when the answer
 * turns, never after shutdown().
 */





























void test_begin_defers_the_client_while_no_socket_can_be_opened() {
    HAL::setCanOpenServerForTest(false);
    NTPComponent ntp;
    TEST_ASSERT_EQUAL(ComponentStatus::Success, ntp.begin());
    TEST_ASSERT_FALSE(clientRunning());

    ntp.loop();
    TEST_ASSERT_FALSE_MESSAGE(clientRunning(), "loop() started it with no interface");
}

void test_loop_starts_the_deferred_client_once_an_interface_appears() {
    HAL::setCanOpenServerForTest(false);
    NTPComponent ntp;
    ntp.begin();

    HAL::setCanOpenServerForTest(true);
    ntp.loop();
    TEST_ASSERT_TRUE(clientRunning());
}

void test_sync_now_is_refused_before_the_client_starts() {
    HAL::setCanOpenServerForTest(false);
    NTPComponent ntp;
    ntp.begin();

    TEST_ASSERT_FALSE(ntp.syncNow());
    TEST_ASSERT_EQUAL_UINT32(0, HAL::NTPImpl::forceSyncCalls());
}

void test_a_deferred_client_stays_stopped_after_shutdown() {
    HAL::setCanOpenServerForTest(false);
    NTPComponent ntp;
    ntp.begin();
    ntp.shutdown();

    HAL::setCanOpenServerForTest(true);
    ntp.loop();
    TEST_ASSERT_FALSE_MESSAGE(clientRunning(), "loop() restarted a client shutdown() stopped");
}

void test_a_running_client_stays_stopped_after_shutdown() {
    NTPComponent ntp;
    ntp.begin();
    ntp.shutdown();
    TEST_ASSERT_FALSE(clientRunning());

    ntp.loop();
    TEST_ASSERT_FALSE_MESSAGE(clientRunning(), "loop() restarted a client shutdown() stopped");
}














































































































