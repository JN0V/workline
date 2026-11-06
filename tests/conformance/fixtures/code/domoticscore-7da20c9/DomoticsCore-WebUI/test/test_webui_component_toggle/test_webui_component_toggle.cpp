
 * @file test_webui_component_toggle.cpp
 * @brief Enabling or disabling a component from the page runs in loop().
 *
 * The route runs on the web server's task. A component's begin() or shutdown(),
 * and the provider registry the loop reads, must not be touched from there.
 */





















    void buildContexts(std::vector<WebUIContext>& contexts) override {
        contexts.push_back(WebUIContext::dashboard("toggle_ctx", "Toggle"));
    }
public:
    String getWebUIName() const override { return "Toggle"; }












    return (start < 0 || end <= start + 3) ? String("") : body.substring(start + 3, end);
}

// What the page reads back: the Toggle entry's "enabled" in GET /api/components.
bool listedEnabled() {
    AsyncWebServerRequest request;
    AsyncWebServer::findRouteAnywhere("/api/components")->handler(&request);
    JsonDocument doc;
    deserializeJson(doc, request.sentBody);
    for (JsonObject c : doc["components"].as<JsonArray>()) {
        if (strcmp(c["name"] | "", "Toggle") == 0) return c["enabled"] | false;
    }
    TEST_FAIL_MESSAGE("Toggle is not listed");
    return false;
}















void test_a_disable_from_the_page_waits_for_loop() {
    WebUIComponent webui;
    webui.begin();
    CountedComponent component;
    ToggleProvider provider;
    webui.registerProviderWithComponent(&provider, &component);

    AsyncWebServerRequest request;
    post("Toggle", "false", request);
    TEST_ASSERT_TRUE_MESSAGE(request.sentBody.indexOf("\"success\":true") >= 0, request.sentBody.c_str());
    TEST_ASSERT_EQUAL_INT_MESSAGE(0, component.shutdowns, "shutdown() ran on the web server's task");
    TEST_ASSERT_TRUE_MESSAGE(listedEnabled(), "the registry was changed on the web server's task");

    webui.loop();
    TEST_ASSERT_EQUAL_INT(1, component.shutdowns);
    TEST_ASSERT_FALSE(listedEnabled());
}

void test_an_enable_from_the_page_waits_for_loop() {
    WebUIComponent webui;
    webui.begin();
    CountedComponent component;
    ToggleProvider provider;
    webui.registerProviderWithComponent(&provider, &component);

    AsyncWebServerRequest off;
    post("Toggle", "false", off);
    webui.loop();

    AsyncWebServerRequest on;
    post("Toggle", "true", on);
    TEST_ASSERT_EQUAL_INT_MESSAGE(0, component.begins, "begin() ran on the web server's task");
    webui.loop();
    TEST_ASSERT_EQUAL_INT(1, component.begins);
    TEST_ASSERT_TRUE(listedEnabled());
}

































