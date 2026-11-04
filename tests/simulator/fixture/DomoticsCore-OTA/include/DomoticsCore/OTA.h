// Over-the-air updates: an upload announced, then its chunks.
#pragma once
#include <cstddef>

namespace DomoticsCore {

class OTAComponent {
public:
    bool beginUpload(size_t announced);
    bool acceptUploadChunk(size_t length);

private:
    size_t announced_ = 0;
    size_t received_ = 0;
};

}  // namespace DomoticsCore
