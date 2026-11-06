







 * printing is on, the SDK's own narration. The sink forwards every character to
 * UART0 the way the core's own does, so taking it costs the serial port nothing.
 *
 * SDK printing is off by default and not by accident: HardwareSerial::begin()
 * calls end(), which calls uart_set_debug(UART_NO) and system_set_os_print(0).
 * Measured on a nodemcuv2 — a full failed join prints nothing at all — so
 * setSdkOutput(true) is what makes those lines exist, on the serial port as well



















































inline bool setSdkOutput(bool enabled) {
    system_set_os_print(enabled ? 1 : 0);
    if (enabled) takeSink();   // an application's setDebugOutput() may have taken the sink
    return true;
}







