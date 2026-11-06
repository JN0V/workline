

















namespace Components {

/**
 * Component registry for managing component lifecycle and dependencies
 * Handles registration, dependency resolution, and coordinated initialization
 */
class ComponentRegistry {








    bool initialized = false;
    // Lifecycle listeners
    std::vector<IComponentLifecycleListener*> listeners;
    // Event bus for cross-component communication
    DomoticsCore::Utils::EventBus eventBus;
    // Core reference for automatic injection to components
    DomoticsCore::Core* core_ = nullptr;
    
    
public:
    /**
     * Register a component with the registry
     * @param component Unique pointer to component instance
     * @return true if registration successful, false if duplicate name
     */
    bool registerComponent(std::unique_ptr<IComponent> component) {
        if (!component) {
            DLOG_E(LOG_CORE, "Cannot register null component");
            return false;
        }
        
        String name(component->metadata.name);
        if (componentMap.find(name) != componentMap.end()) {
            DLOG_E(LOG_CORE, "Component '%s' already registered", component->metadata.name);
            return false;
        }
        
        IComponent* ptr = component.get();
        
        // Inject registry immediately for lazy Core access
        ptr->__dc_setRegistry(this);
        
        componentMap[name] = ptr;
        components.push_back(std::move(component));
        
        DLOG_I(LOG_CORE, "Registered component: %s v%s", 
               ptr->metadata.name, ptr->metadata.version);
        // Notify listeners about addition
        for (auto* l : listeners) {
            if (l) l->onComponentAdded(ptr);
        }
        return true;
    }

























            }
            
            DLOG_I(LOG_CORE, "Initializing component: %s", component->metadata.name);
            // OBS-3: a death inside this begin() names the component, not "idle".
            FlightRecorder::instance().setPhase(static_cast<uint16_t>(FlightRecorder::PHASE_INIT | initIndex));
            
            // Provide framework services (EventBus, Core) to the component before begin()
            if (component) {
                component->__dc_setEventBus(&eventBus);
                component->__dc_setCore(core_);
            }

            ComponentStatus status = component->begin();
            if (status != ComponentStatus::Success) {
                DLOG_E(LOG_CORE, "Failed to initialize component %s: %s", 
                       component->metadata.name, statusToString(status));







































    /**
     * Call loop() on all active components
     */
    /** Name of the component at 1-based initialization index, or nullptr (phase marker decoding, OBS-3). */
    const char* componentNameAtInitIndex(size_t index1) const {
        if (index1 == 0 || index1 > initializationOrder.size()) return nullptr;
        return initializationOrder[index1 - 1]->metadata.name;
    }

    void loopAll() {
        if (!initialized) return;
        
        // OBS-3: the phase marker names who was running when the device died.
        FlightRecorder& rec = FlightRecorder::instance();
        uint16_t index = 0;
        for (auto* component : initializationOrder) {
            ++index;
            if (component->isActive()) {
                rec.setPhase(index < FlightRecorder::PHASE_EVENT_DISPATCH ? index : FlightRecorder::PHASE_EVENT_DISPATCH - 1);
                component->loop();
            }
        }
        rec.setPhase(FlightRecorder::PHASE_EVENT_DISPATCH);
        eventBus.poll();
        rec.setPhase(FlightRecorder::PHASE_IDLE);
    }
    
    /**
     * Shutdown all components in reverse order
     */
    void shutdownAll() {
        if (!initialized) return;

        // Publish shutdown start event and dispatch immediately
        // so listeners can react before components are shut down
        eventBus.publish(Events::EVENT_SHUTDOWN_START);  // BUG-30: see EVENT_SYSTEM_READY above
        eventBus.poll();

        // Shutdown in reverse order
        for (auto it = initializationOrder.rbegin(); it != initializationOrder.rend(); ++it) {
            auto* component = *it;
            if (component->isActive()) {
                DLOG_I(LOG_CORE, "Shutting down component: %s", component->metadata.name);
                ComponentStatus status = component->shutdown();
                if (status != ComponentStatus::Success) {
                    DLOG_W(LOG_CORE, "Component %s shutdown warning: %s", 
                           component->metadata.name, statusToString(status));
                }
                // Clean up EventBus subscriptions for this component
                eventBus.unsubscribeOwner(component);
                component->setActive(false);
            }
        }
        
        initialized = false;
        DLOG_I(LOG_CORE, "All components shut down");
    }

    /**
     * Remove a component by name at runtime. Will shutdown the component and notify listeners.
     */
    bool removeComponent(const String& name) {
        auto itMap = componentMap.find(name);
        if (itMap == componentMap.end()) return false;
        IComponent* comp = itMap->second;
        // Shutdown if active
        if (comp->isActive()) {
            DLOG_I(LOG_CORE, "Shutting down component (remove): %s", name.c_str());
            comp->shutdown();
            // Clean up EventBus subscriptions for this component
            eventBus.unsubscribeOwner(comp);
            comp->setActive(false);
        }
        // Notify listeners
        for (auto* l : listeners) {
            if (l) l->onComponentRemoved(comp);
        }
        // Erase from containers
        componentMap.erase(itMap);
        // Remove from initialization order
        initializationOrder.erase(std::remove(initializationOrder.begin(), initializationOrder.end(), comp), initializationOrder.end());
        initializationOrder.shrink_to_fit();
        // Remove from components vector
        for (auto it = components.begin(); it != components.end(); ++it) {
            if (it->get() == comp) {
                components.erase(it);
                components.shrink_to_fit();
                break;
            }
        }
        DLOG_I(LOG_CORE, "Component removed: %s", name.c_str());
        return true;
    }


































    void removeListener(IComponentLifecycleListener* listener) {
        listeners.erase(std::remove(listeners.begin(), listeners.end(), listener), listeners.end());
        listeners.shrink_to_fit();
    }























































































































