








 *
 * Usage:
 * @code
 * // In setup(), after all components initialized:
 * MemoryManager::instance().detectProfile();
 *
 * // Query profile anywhere:














 * @brief Memory profile levels based on available heap
 */
enum class MemoryProfile {
    FULL,       ///< > 30KB free: All features enabled, max buffers
    STANDARD,   ///< 15-30KB free: Moderate reductions
    MINIMAL,    ///< 8-15KB free: Economy mode, reduced features
    CRITICAL    ///< < 8KB free: Emergency mode, minimal operation









































    /**
     * @brief Detect memory profile based on current free heap
     *
     * Call this once in setup() before components are initialized,
     * to get an accurate picture of available runtime memory.
     * Can also be called implicitly via getProfile().
     *
     * @return Detected profile
     */
    MemoryProfile detectProfile() const {
        uint32_t freeHeap = HAL::getFreeHeap();
        heapAtBoot_ = freeHeap;

        if (freeHeap >= thresholds_.fullMin) {
            profile_ = MemoryProfile::FULL;
        } else if (freeHeap >= thresholds_.standardMin) {
            profile_ = MemoryProfile::STANDARD;
        } else if (freeHeap >= thresholds_.minimalMin) {
            profile_ = MemoryProfile::MINIMAL;
        } else {
            profile_ = MemoryProfile::CRITICAL;
        }

        detected_ = true;
        return profile_;
    }

































































































