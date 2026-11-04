#include <unity.h>
#include <DomoticsCore/LEDWebUI.h>

using namespace DomoticsCore;

void test_select_in_range(void) {
    LEDComponent led;
    led.addLED("kitchen");
    LEDWebUI ui(&led);
    TEST_ASSERT_TRUE(ui.handleSelect(0));
}

int main() {
    UNITY_BEGIN();
    RUN_TEST(test_select_in_range);
    return UNITY_END();
}
