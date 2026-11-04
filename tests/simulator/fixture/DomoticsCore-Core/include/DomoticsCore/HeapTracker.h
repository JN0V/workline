// Heap measures for the ESP8266 suites.
#pragma once
#include <cstddef>

namespace DomoticsCore {

class HeapTracker {
public:
    void checkpoint(size_t free) { start_ = free; }
    long getDelta(size_t free) const { return static_cast<long>(start_) - static_cast<long>(free); }

private:
    size_t start_ = 0;
};

}  // namespace DomoticsCore
