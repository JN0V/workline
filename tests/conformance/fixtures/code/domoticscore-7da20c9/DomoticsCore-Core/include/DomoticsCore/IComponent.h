




















































    /**
     * Initialize the component
     * Called during Core.begin() after dependencies are resolved
     * @return ComponentStatus indicating success or specific error
     */
    virtual ComponentStatus begin() = 0;
    






    /**
     * Shutdown the component
     * Called during Core.shutdown() or component removal
     * @return ComponentStatus indicating success or specific error
     */
    virtual ComponentStatus shutdown() = 0;
    





































































    /**
     * Optional: Called after ALL components (including built-ins) are ready
     * Use this for late initialization that depends on other components
     * All components declared in getDependencies() are guaranteed available here
     * 
     * Lifecycle order:
     * 1. begin() - Internal initialization only (GPIO, state, etc.)
     * 2. afterAllComponentsReady() - Dependency setup (can access other components safely)
     * 3. loop() - Normal operation
     * 
     * Example:
     * ```cpp
     * ComponentStatus begin() override {
     *     pinMode(PIN, INPUT);  // Internal init only
     *     return ComponentStatus::Success;
     * }





























































































