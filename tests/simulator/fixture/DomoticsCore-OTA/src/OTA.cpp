#include "DomoticsCore/OTA.h"

namespace DomoticsCore {

bool OTAComponent::beginUpload(size_t announced) {
    announced_ = announced;
    received_ = 0;
    return announced > 0;
}

bool OTAComponent::acceptUploadChunk(size_t length) {
    received_ += length;
    return received_ <= announced_;
}

}  // namespace DomoticsCore
