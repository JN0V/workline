// The Storage component: a named namespace of preferences.
#pragma once
#include <string>

namespace DomoticsCore {

struct ComponentMetadata {
    const char* name = "Storage";
};

class StorageComponent {
public:
    explicit StorageComponent(const std::string& componentName = "") {
        if (!componentName.empty()) {
            nameStr_ = componentName;
            metadata.name = nameStr_.c_str();
        }
    }

    ComponentMetadata metadata;

private:
    std::string nameStr_;
};

}  // namespace DomoticsCore
