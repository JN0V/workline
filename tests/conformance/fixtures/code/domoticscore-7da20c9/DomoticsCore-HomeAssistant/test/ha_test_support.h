































static void startLogCapture() {
    if (g_captureActive) LoggerCallbacks::removeCallback(g_captureId);
    g_capturedWarn = "";
    g_capturedError = "";
    g_capturedInfo = "";
    g_captureId = LoggerCallbacks::addCallback(
        [](LogLevel level, const char* tag, const char* message) {
            if (strcmp(tag, LOG_HA) != 0) return;
            if (level == LOG_LEVEL_WARN) {
                g_capturedWarn += message;
                g_capturedWarn += "\n";
            } else if (level == LOG_LEVEL_ERROR) {
                g_capturedError += message;
                g_capturedError += "\n";
            } else if (level == LOG_LEVEL_INFO) {
                g_capturedInfo += message;
                g_capturedInfo += "\n";
            }
        });
    g_captureActive = true;
}
















