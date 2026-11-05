/**
 * @file test_ha_dropped_link.cpp
 * @brief HomeAssistant component: a dropped link, at the seam between MQTT and HomeAssistant.
 */

#include <unity.h>
#include "../ha_test_support.h"

// ============================================================================
// A dropped link, at the seam between MQTT and HomeAssistant
// ============================================================================

// isReady() gates every publish this component makes, and it can only fall if
// MQTT says the link is gone. The real component is driven here, not
// simulateMqttConnect(): the whole point is that the event comes from a loop()
// noticing a client that went down on its own.
struct DroppedLinkFixture {
    Core core;
    Components::MQTTComponent* mqtt = nullptr;
    HomeAssistantComponent* ha = nullptr;

    DroppedLinkFixture() {
        HAL::WiFiImpl::setConnectedForTest(true);
        Components::MQTTConfig mcfg;
        mcfg.clientId = "ESP32-bug44";
        mcfg.broker = "192.0.2.1";
        mcfg.autoReconnect = false;        // no reconnection to muddy the reading
        core.addComponent(std::make_unique<Components::MQTTComponent>(mcfg));

        HAConfig hcfg;
        HA::setField(hcfg.nodeId, "test_node", sizeof(hcfg.nodeId));
        core.addComponent(std::make_unique<HomeAssistantComponent>(hcfg));
        core.begin();

        mqtt = core.getComponent<Components::MQTTComponent>("MQTT");
        ha = core.getComponent<HomeAssistantComponent>("HomeAssistant");
        TEST_ASSERT_NOT_NULL_MESSAGE(mqtt, "the fixture has no MQTT component");
        TEST_ASSERT_NOT_NULL_MESSAGE(ha, "the fixture has no HomeAssistant component");
        TEST_ASSERT_TRUE_MESSAGE(mqtt->connect(), "the fixture never connected");
        drain();
        TEST_ASSERT_TRUE_MESSAGE(ha->isReady(), "the fixture never came up ready");
    }

    ~DroppedLinkFixture() {
        core.shutdown();
        HAL::WiFiImpl::setConnectedForTest(false);
    }

    void drain() { for (int i = 0; i < 5; i++) core.loop(); }

    // The broker drops the link: the client goes down under the component.
    void dropTheLink() {
        auto* client = mqtt->getClientForTest();
        TEST_ASSERT_NOT_NULL_MESSAGE(client, "nothing to drop: no client was built");
        client->disconnect();
        drain();
    }
};

void test_a_dropped_link_takes_home_assistant_out_of_ready() {
    DroppedLinkFixture f;
    f.dropTheLink();

    TEST_ASSERT_FALSE_MESSAGE(f.ha->isReady(),
        "isReady() answers true over a dead link, so every publish guard in this "
        "component stops guarding");
}

// isReady() is the conjunction of two members, so it falls if either does. The
// guards the entry is about read mqttConnected alone: pin that one through a
// publish, or an edit that clears the other member keeps the test above green
// while every guard stays open.
void test_a_state_published_during_an_outage_reaches_nothing() {
    DroppedLinkFixture f;
    f.ha->addSensor("uptime", "Uptime", "s");
    f.drain();
    TEST_ASSERT_EQUAL_UINT32_MESSAGE(0, (uint32_t)f.mqtt->getQueuedMessageCount(),
        "the fixture starts with a drained queue or this proves nothing");

    f.dropTheLink();
    f.ha->publishState("uptime", String(42));
    f.drain();

    TEST_ASSERT_EQUAL_UINT32_MESSAGE(0, (uint32_t)f.mqtt->getQueuedMessageCount(),
        "the state was handed to MQTT during the outage: the mqttConnected guard "
        "is still open");
}

// The fall must be transient. mqtt/disconnected now fires where it never did,
// so a component that does not re-arm on the reconnection would go mute for the
// life of the process — a worse failure than the one being fixed.
void test_home_assistant_is_ready_again_after_the_link_returns() {
    DroppedLinkFixture f;
    f.dropTheLink();
    TEST_ASSERT_FALSE(f.ha->isReady());

    TEST_ASSERT_TRUE_MESSAGE(f.mqtt->connect(), "the fixture never reconnected");
    f.drain();

    TEST_ASSERT_TRUE_MESSAGE(f.ha->isReady(),
        "the link came back and HomeAssistant stayed mute");
}

// Over a dead link shutdown() has nothing to deliver: it must not queue the
// offline availability and the removals on the EventBus.
void test_shutdown_during_an_outage_queues_no_mqtt_publish() {
    DroppedLinkFixture f;
    f.ha->addSensor("uptime", "Uptime", "s");
    f.drain();
    f.dropTheLink();

    int publishes = 0;
    f.core.on<MQTTPublishEvent>(DomoticsCore::MQTTEvents::EVENT_PUBLISH,
        [&](const MQTTPublishEvent&) { publishes++; });

    f.ha->shutdown();
    f.drain();

    TEST_ASSERT_EQUAL_INT_MESSAGE(0, publishes,
        "shutdown() queued MQTT publishes over a dropped link");
}

// The connected case keeps its farewell: offline availability and removals.
void test_shutdown_while_connected_still_emits_the_farewell() {
    DroppedLinkFixture f;
    f.ha->addSensor("uptime", "Uptime", "s");
    f.drain();

    int removals = 0;
    int publishes = 0;
    f.core.on<MQTTPublishEvent>(DomoticsCore::MQTTEvents::EVENT_PUBLISH,
        [&](const MQTTPublishEvent& ev) {
            publishes++;
            if (strstr(ev.topic, "/config") && ev.payload[0] == '\0') removals++;
        });

    f.ha->shutdown();
    f.drain();

    TEST_ASSERT_EQUAL_INT_MESSAGE(1, removals, "shutdown() left the entity in Home Assistant");
    TEST_ASSERT_TRUE_MESSAGE(publishes >= 2, "shutdown() did not publish the offline availability");
}

static int runAllTests() {
    UNITY_BEGIN();

    // A dropped link, at the seam between MQTT and HomeAssistant
    RUN_TEST(test_a_dropped_link_takes_home_assistant_out_of_ready);
    RUN_TEST(test_a_state_published_during_an_outage_reaches_nothing);
    RUN_TEST(test_home_assistant_is_ready_again_after_the_link_returns);
    RUN_TEST(test_shutdown_during_an_outage_queues_no_mqtt_publish);
    RUN_TEST(test_shutdown_while_connected_still_emits_the_farewell);

    return UNITY_END();
}

int main(int argc, char** argv) { return runAllTests(); }
