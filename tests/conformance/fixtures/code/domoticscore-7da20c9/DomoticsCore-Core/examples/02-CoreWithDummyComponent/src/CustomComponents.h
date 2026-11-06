




























































    ComponentStatus begin() override {
        DLOG_I(LOG_APP, "TestComponent '%s' initializing...", componentName.c_str());
        
        // Simulate some initialization work
        delay(50);
        
        counter = 0;
        setStatus(ComponentStatus::Success);
        DLOG_I(LOG_APP, "TestComponent '%s' initialized successfully", componentName.c_str());
        return ComponentStatus::Success;
    }
    
    void loop() override {
        // Heartbeat logging
        if (heartbeatTimer.isReady()) {
            DLOG_I(LOG_APP, "TestComponent '%s' heartbeat - counter: %d, uptime: %lu ms", 
                   componentName.c_str(), counter, millis());
        }
        
        // Simulate periodic work
        if (simulateWork && workTimer.isReady()) {
            counter++;
            DLOG_D(LOG_APP, "TestComponent '%s' doing work iteration %d", 
                   componentName.c_str(), counter);
            
            // Simulate different work patterns
            if (counter % 10 == 0) {
                DLOG_I(LOG_APP, "TestComponent '%s' milestone reached: %d iterations", 
                       componentName.c_str(), counter);
            }
            
            if (counter % 25 == 0) {
                DLOG_W(LOG_APP, "TestComponent '%s' warning: high iteration count (%d)", 
                       componentName.c_str(), counter);
            }
            
            // Check max iterations limit
            if (config.maxIterations > 0 && counter >= config.maxIterations) {
                simulateWork = false;
                DLOG_I(LOG_APP, "TestComponent '%s' reached max iterations (%d)", 
                       componentName.c_str(), config.maxIterations);
            }
        }
    }
    
    ComponentStatus shutdown() override {
        DLOG_I(LOG_APP, "TestComponent '%s' shutting down...", componentName.c_str());
        simulateWork = false;
        setStatus(ComponentStatus::Success);
        DLOG_I(LOG_APP, "TestComponent '%s' shutdown complete - final counter: %d", 
               componentName.c_str(), counter);
        return ComponentStatus::Success;
    }








































































    ComponentStatus begin() override {
        DLOG_I(LOG_APP, "LEDBlinker initializing on pin %d...", ledPin);
        
        // Initialize component metadata
        metadata.name = "LEDBlinker";
        metadata.version = "1.0.0";
        metadata.author = "DomoticsCore Example";
        metadata.description = "LED blinker component for hardware demonstration";
        
        pinMode(ledPin, OUTPUT);
        digitalWrite(ledPin, HAL::ledBuiltinOff());  // Turn LED off initially
        ledState = false;
        setStatus(ComponentStatus::Success);
        DLOG_I(LOG_APP, "LEDBlinker initialized successfully");
        return ComponentStatus::Success;
    }









    ComponentStatus shutdown() override {
        DLOG_I(LOG_APP, "LEDBlinker shutting down...");
        digitalWrite(ledPin, HAL::ledBuiltinOff());
        blinkEnabled = false;
        setStatus(ComponentStatus::Success);
        return ComponentStatus::Success;
    }









































