// The OTA component's web page: the upload handler.
#pragma once
#include <string>
#include "DomoticsCore/OTA.h"

namespace DomoticsCore {

struct UploadState {
    bool rejected = false;
};

class OTAWebUI {
public:
    explicit OTAWebUI(OTAComponent* ota) : ota_(ota) {}

    std::string getWebUIVersion() const { return "1.2.0"; }

    // The upload handler: a refused begin marks the upload rejected.
    int handleUpload(size_t announced, size_t chunk) {
        if (!ota_->beginUpload(announced)) {
            uploadState.rejected = true;
            return 400;
        }
        ota_->acceptUploadChunk(chunk);
        return uploadState.rejected ? 400 : 200;
    }

    UploadState uploadState;

private:
    OTAComponent* ota_;
};

}  // namespace DomoticsCore
