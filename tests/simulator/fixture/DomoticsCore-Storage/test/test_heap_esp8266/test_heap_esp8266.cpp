#include <unity.h>
#include <DomoticsCore/HeapTracker.h>
#include <string>

// The heap suite: each test opens a namespace of its own.
struct OpenStorage {
    explicit OpenStorage(const std::string& ns) : name(ns) {}
    std::string name;
};

void test_namespace_opens(void) {
    OpenStorage s("heaptest");
    TEST_ASSERT_TRUE(s.name == "heaptest");
}

int main() {
    UNITY_BEGIN();
    RUN_TEST(test_namespace_opens);
    return UNITY_END();
}
