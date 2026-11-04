#include <unity.h>
#include <DomoticsCore/OTAWebUI.h>

using namespace DomoticsCore;

void test_empty_upload_refused(void) {
    OTAComponent ota;
    OTAWebUI ui(&ota);
    TEST_ASSERT_EQUAL(ui.handleUpload(0, 0), 400);
}

int main() {
    UNITY_BEGIN();
    RUN_TEST(test_empty_upload_refused);
    return UNITY_END();
}
