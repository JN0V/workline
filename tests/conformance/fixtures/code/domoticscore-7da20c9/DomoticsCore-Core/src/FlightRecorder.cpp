






















































































































































void FlightRecorder::begin(bool holdUntilAcknowledged) {
    if (begun_) return;
    begun_ = true;
    hold_ = holdUntilAcknowledged;
    lastTickMs_ = lastSlowMs_ = HAL::Platform::getMillis();
    restartHookInstalled_ = HAL::Platform::installRestartHook(&markOursTrampoline);
    resetReason_ = HAL::Platform::getResetReason();
    // A cliff or a crash inside the first interval measures against this.
    lastTickFree_ = runMin_ = HAL::Platform::getAllocatableFreeHeap();

    HAL::Platform::rtcRead(0, previous_.w, FlightRecord::WORDS);
    // Layout 1 (before OBS-4) is read with its own crc rule and rewritten as
    // layout 2: the first boot after that upgrade keeps its death.
    const uint32_t layout = previous_.layout();
    const bool magicOk = previous_.w[FlightRecord::W_MAGIC] == FlightRecord::MAGIC
                      && (layout == FlightRecord::LAYOUT || layout == FlightRecord::LAYOUT_V1);
    if (!magicOk) {
        startFresh(0);
        writeAll(current_);
    } else {
        const uint32_t flags = previous_.flags();
        torn_ = previous_.w[FlightRecord::W_CRC] != previous_.bodyCrcForLayout(layout);

        if (flags & FlightRecord::UNPERSISTED) {
            // Died again before the last promotion was acknowledged: the record
            // in RTC is still the first death, promoted for the reason stored then.
            promotion_ = static_cast<Promotion>((flags & FlightRecord::PROMO_MASK) >> FlightRecord::PROMO_SHIFT);
            if (promotion_ == Promotion::None) promotion_ = Promotion::CrashCallback;
        } else {
            const HAL::Platform::ResetReason reason = resetReason_;
            if (flags & FlightRecord::CALLBACK_RAN) {
                promotion_ = Promotion::CrashCallback;
            } else if (HAL::Platform::wasUnexpectedReset(reason)) {
                promotion_ = Promotion::UnexpectedReset;
            } else if (reason == HAL::Platform::ResetReason::Software && !(flags & FlightRecord::OURS)) {
                promotion_ = Promotion::UnownedSoftwareReset;
            }
        }

        startFresh(previous_.bootSequence() + 1);
        if (hasPromotedRecord()) {
            uint32_t meta = previous_.w[FlightRecord::W_META] | FlightRecord::UNPERSISTED
                          | (static_cast<uint32_t>(promotion_) << FlightRecord::PROMO_SHIFT)
                          | (torn_ ? FlightRecord::TORN : 0u);
            previous_.w[FlightRecord::W_META] = meta;
            HAL::Platform::rtcStoreWord(FlightRecord::W_META, meta);   // outside the crc: the body stays verifiable
            // RTC keeps the death until acknowledge()
        } else {
            writeAll(current_);
        }
    }
    // Registered last: a failure on another task during the lines above would
    // write a group that startFresh()/writeAll() then erase (OBS-4).
    failedAllocHookInstalled_ = HAL::Platform::installFailedAllocHook(&noteFailedAllocTrampoline);
}


















































































































































































































































































