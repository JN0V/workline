// A few of Unity's macros, enough for the simulator's suites: each test
// prints its name and PASS or FAIL, as Unity's runner does.
#pragma once
#include <cstdio>
#include <cstring>

static int unity_failed_ = 0;
static int unity_failures_ = 0;

#define TEST_ASSERT_TRUE(c) do { if (!(c)) { unity_failed_ = 1; std::printf("  assert: %s\n", #c); return; } } while (0)
#define TEST_ASSERT_FALSE(c) TEST_ASSERT_TRUE(!(c))
#define TEST_ASSERT_EQUAL(a, b) TEST_ASSERT_TRUE((a) == (b))
#define TEST_ASSERT_EQUAL_STRING(a, b) TEST_ASSERT_TRUE(std::strcmp((a), (b)) == 0)
#define UNITY_BEGIN() (unity_failures_ = 0)
#define RUN_TEST(f) do { unity_failed_ = 0; f(); std::printf("%s:%s\n", #f, unity_failed_ ? "FAIL" : "PASS"); unity_failures_ += unity_failed_; } while (0)
#define UNITY_END() (unity_failures_)
