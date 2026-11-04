// A Home Assistant entity: its state, published as a payload.
#pragma once
#include <string>

namespace DomoticsCore {

class HAEntity {
public:
    virtual ~HAEntity() = default;
    void publishState(const std::string& payload) { last_ = payload; }
    void publishStateJson(const std::string& json) { last_ = json; }
    const std::string& lastPayload() const { return last_; }

private:
    std::string last_;
};

}  // namespace DomoticsCore
