// The remote console: telnet clients, each with an id.
#pragma once
#include <cstdint>
#include <vector>

namespace DomoticsCore {

class RemoteConsoleComponent {
public:
    // loop(): a client accepted gets the next id.
    void loop(bool accepted) {
        if (accepted) clients_.push_back(nextClientId++);
    }
    const std::vector<uint32_t>& clients() const { return clients_; }

private:
    uint32_t nextClientId = 1;
    std::vector<uint32_t> clients_;
};

}  // namespace DomoticsCore
