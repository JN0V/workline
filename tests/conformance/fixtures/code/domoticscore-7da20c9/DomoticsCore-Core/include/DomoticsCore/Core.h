

























 *
 * Manages configuration, logging, and registration/lifecycle of components stored in the
 * `Components::ComponentRegistry`. Provides convenience helpers for accessing components by name
 * or type and drives their `begin()`, `loop()`, and `shutdown()` methods.
 */
class Core {
private:
    CoreConfig config;
    bool initialized;
    // The two once-a-minute lines of loop() (LO-5/BUG-36 drops, OBS-4 failures): per instance, not static.
    uint32_t lastDropsLogged_ = 0, lastFailsLogged_ = 0;
    unsigned long lastDropLog_ = 0, lastFailLog_ = 0;
    Components::ComponentRegistry componentRegistry;




     */
    Core();
    /**
     * @brief Clean up and ensure `shutdown()` has been called.
     */
    ~Core();
    
    // Core lifecycle
    /**
     * @brief Initialize logging, store configuration, and start all registered components.
     * @param cfg Configuration block (device name, ID, log level).
     * @return true on success, false if any component fails to initialize.
     */
    bool begin(const CoreConfig& cfg = CoreConfig());
    /**
     * @brief Drive the `loop()` method of each registered component.
     */
    void loop();
    /**
     * @brief Stop all components and release resources.
     */
    void shutdown();
    
    // Configuration access
    CoreConfig getConfiguration() const { return config; }























































































































































