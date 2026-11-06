


 *
 * On ESP32 lwIP exists from the first network interface on; a server begun
 * before that aborts the firmware. The component asks HAL::canOpenServer() in
 * begin() and opens from loop() when the answer turns, never after shutdown().
 */

#include <unity.h>













































void test_a_deferred_server_stays_closed_after_shutdown() {
    HAL::setCanOpenServerForTest(false);
    WebUIComponent webui;
    webui.begin();
    webui.shutdown();

    TEST_ASSERT_EQUAL_INT_MESSAGE(0, AsyncWebServer::last()->endCalls, "end() reached a server never begun");

    HAL::setCanOpenServerForTest(true);
    webui.loop();
    TEST_ASSERT_FALSE_MESSAGE(serverRunning(), "loop() reopened a server shutdown() closed");
}

void test_an_open_server_stays_closed_after_shutdown() {
    WebUIComponent webui;
    webui.begin();
    webui.shutdown();
    TEST_ASSERT_FALSE(serverRunning());

    webui.loop();
    TEST_ASSERT_FALSE_MESSAGE(serverRunning(), "loop() reopened a server shutdown() closed");
}










