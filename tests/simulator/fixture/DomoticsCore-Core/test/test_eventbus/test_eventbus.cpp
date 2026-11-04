#include <unity.h>
#include <DomoticsCore/EventBus.h>

using namespace DomoticsCore;

void test_subscribe_receives(void) {
    EventBus bus;
    std::string got;
    bus.subscribe("a", [&](const std::string& p) { got = p; });
    bus.publish("a", "on");
    TEST_ASSERT_TRUE(got == "on");
}

int main() {
    UNITY_BEGIN();
    RUN_TEST(test_subscribe_receives);
    return UNITY_END();
}
