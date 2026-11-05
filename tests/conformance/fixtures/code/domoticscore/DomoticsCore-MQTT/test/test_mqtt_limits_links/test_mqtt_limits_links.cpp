/**
 * @file test_mqtt_limits_links.cpp
 * @brief MQTT component: configured limits, publishNow(), and a link the
 *        broker or the network drops.
 */

#include <unity.h>
#include <memory>
#include <string>
#include <vector>
#include <DomoticsCore/Core.h>
#include <DomoticsCore/MQTT.h>
#include <DomoticsCore/MQTTEvents.h>
#include <DomoticsCore/Testing/HeapTracker.h>
using namespace DomoticsCore;
using namespace DomoticsCore::Components;

void setUp() {}
void tearDown() {
    HAL::WiFiImpl::setConnectedForTest(false);
}

// ============================================================================
// Config Limit Enforcement Tests
// ============================================================================

void test_mqtt_queue_rejects_when_full() {
    MQTTConfig cfg;
    cfg.broker = "test.local";
    cfg.maxQueueSize = 5;
    MQTTComponent mqtt(cfg);
    mqtt.begin();
    // Not connected → messages get queued
    for (int i = 0; i < 5; i++) {
        TEST_ASSERT_TRUE(mqtt.publish("topic", String(i)));
    }
    // 6th should be rejected
    TEST_ASSERT_FALSE(mqtt.publish("topic", "overflow"));
    mqtt.shutdown();
}

void test_mqtt_queue_unlimited_when_zero() {
    MQTTConfig cfg;
    cfg.broker = "test.local";
    cfg.maxQueueSize = 0;
    cfg.publishRateLimit = 0; // disable rate limit to test queue only
    MQTTComponent mqtt(cfg);
    mqtt.begin();
    // No count limit: only the platform's byte budget stops the queue.
    size_t accepted = 0;
    while (accepted < 10000 && mqtt.publish("topic", String((int)accepted))) accepted++;
    TEST_ASSERT_GREATER_THAN(HAL::MQTT::kQueueByteBudget - 100, mqtt.getQueuedBytes());
    TEST_ASSERT_EQUAL_UINT32_MESSAGE(1, mqtt.getStatistics().publishErrors, "refused once, by the byte budget");
    mqtt.shutdown();
}

void test_mqtt_subscribe_rejects_at_limit() {
    MQTTConfig cfg;
    cfg.broker = "test.local";
    cfg.maxSubscriptions = 3;
    MQTTComponent mqtt(cfg);
    mqtt.begin();
    TEST_ASSERT_TRUE(mqtt.subscribe("topic/1"));
    TEST_ASSERT_TRUE(mqtt.subscribe("topic/2"));
    TEST_ASSERT_TRUE(mqtt.subscribe("topic/3"));
    TEST_ASSERT_FALSE(mqtt.subscribe("topic/4"));
    mqtt.shutdown();
}

void test_mqtt_subscribe_unlimited_when_zero() {
    MQTTConfig cfg;
    cfg.broker = "test.local";
    cfg.maxSubscriptions = 0;
    MQTTComponent mqtt(cfg);
    mqtt.begin();
    for (int i = 0; i < 100; i++) {
        TEST_ASSERT_TRUE(mqtt.subscribe("topic/" + String(i)));
    }
    mqtt.shutdown();
}

void test_mqtt_rate_limit_enforced() {
    // The limit governs what goes out on the wire, so it is only observable
    // while connected. Offline, every message is queued and bounded by
    // maxQueueSize instead — nothing is being sent, so there is no rate to cap.
    //
    // This test used to assert the opposite: that the third publish returned
    // false while disconnected. That was the discard this test now guards against, and the
    // counter it relied on advanced on queueing, charging each deferred message
    // twice — once on the way in, once when the queue drained it.
    MQTTConfig cfg;
    cfg.broker = "test.local";
    cfg.publishRateLimit = 3;
    cfg.maxQueueSize = 0;  // unlimited queue — isolate the rate limit
    MQTTComponent mqtt(cfg);
    mqtt.begin();

    HAL::WiFiImpl::setConnectedForTest(true);
    mqtt.connect();
    TEST_ASSERT_TRUE(mqtt.isConnected());

    const uint32_t sentBefore = mqtt.getStatistics().publishCount;
    for (int i = 0; i < 8; i++) {
        TEST_ASSERT_TRUE(mqtt.publish("topic", String(i)));
    }

    // Exactly the limit reached the broker...
    TEST_ASSERT_EQUAL_UINT32(3, mqtt.getStatistics().publishCount - sentBefore);
    // ...and the rest is waiting, not gone. Before, the five would have
    // been discarded, which is how a device lost every entity it declared last.
    TEST_ASSERT_EQUAL_UINT32(5, mqtt.getQueuedMessageCount());

    mqtt.shutdown();
    HAL::WiFiImpl::setConnectedForTest(false);
}

void test_mqtt_rate_limited_messages_drain_next_window() {
    // Deferring is only useful if the queue actually moves. processMessageQueue()
    // runs on every connected loop(), so the backlog leaves over the following
    // seconds without changing the sustained rate.
    //
    // The only slow test in this suite: native time is real time
    // (Platform_Stub.h uses steady_clock), so the tumbling window cannot be
    // fast-forwarded.
    MQTTConfig cfg;
    cfg.broker = "test.local";
    cfg.publishRateLimit = 3;
    cfg.maxQueueSize = 0;
    MQTTComponent mqtt(cfg);
    mqtt.begin();

    HAL::WiFiImpl::setConnectedForTest(true);
    mqtt.connect();

    for (int i = 0; i < 8; i++) mqtt.publish("topic", String(i));
    TEST_ASSERT_EQUAL_UINT32(5, mqtt.getQueuedMessageCount());

    HAL::Platform::delayMs(1100);  // let the window turn over
    mqtt.loop();

    // Three more go out, and the loop stops at the limit rather than re-queueing
    // what it is iterating over.
    TEST_ASSERT_EQUAL_UINT32(2, mqtt.getQueuedMessageCount());

    HAL::Platform::delayMs(1100);
    mqtt.loop();
    TEST_ASSERT_EQUAL_UINT32(0, mqtt.getQueuedMessageCount());

    mqtt.shutdown();
    HAL::WiFiImpl::setConnectedForTest(false);
}

void test_mqtt_rate_limited_queue_still_bounded() {
    // Deferring must not turn a bounded queue into an unbounded one. Overflow is
    // still a loss, and still counted.
    MQTTConfig cfg;
    cfg.broker = "test.local";
    cfg.publishRateLimit = 2;
    cfg.maxQueueSize = 3;
    MQTTComponent mqtt(cfg);
    mqtt.begin();

    HAL::WiFiImpl::setConnectedForTest(true);
    mqtt.connect();

    const uint32_t errorsBefore = mqtt.getStatistics().publishErrors;
    // 2 sent, next 3 queued, the rest refused.
    for (int i = 0; i < 5; i++) TEST_ASSERT_TRUE(mqtt.publish("topic", String(i)));
    TEST_ASSERT_FALSE(mqtt.publish("topic", "overflow"));

    TEST_ASSERT_EQUAL_UINT32(3, mqtt.getQueuedMessageCount());
    TEST_ASSERT_GREATER_THAN_UINT32(errorsBefore, mqtt.getStatistics().publishErrors);

    mqtt.shutdown();
    HAL::WiFiImpl::setConnectedForTest(false);
}

void test_mqtt_rate_limit_unlimited_when_zero() {
    MQTTConfig cfg;
    cfg.broker = "test.local";
    cfg.publishRateLimit = 0;
    cfg.maxQueueSize = 0;
    MQTTComponent mqtt(cfg);
    mqtt.begin();
    for (int i = 0; i < 50; i++) {
        TEST_ASSERT_TRUE(mqtt.publish("topic", String(i)));
    }
    mqtt.shutdown();
}


// ============================================================================
// publishNow — no queue, no String
// ============================================================================

void test_an_oversized_queued_message_is_dropped_and_the_queue_keeps_draining() {
    // PubSubClient refuses a packet over its buffer on every attempt, and
    // processMessageQueue() broke on the false without erasing — the message was
    // retried on every loop() and everything queued behind it waited forever.
    MQTTConfig cfg;
    cfg.broker = "test.local";
    cfg.maxQueueSize = 0;
    MQTTComponent mqtt(cfg);
    mqtt.begin();
    // Offline: three messages queue, the middle one over the stub's 1024-byte buffer.
    TEST_ASSERT_TRUE(mqtt.publish("q/first", "1"));
    std::string big(1100, 'x');
    TEST_ASSERT_TRUE(mqtt.publish("q/oversized", String(big.c_str())));
    TEST_ASSERT_TRUE(mqtt.publish("q/last", "3"));
    HAL::WiFiImpl::setConnectedForTest(true);
    mqtt.connect();
    mqtt.loop();
    auto* stub = static_cast<HAL::MQTT::MQTTClientImpl*>(mqtt.getClientForTest());
    TEST_ASSERT_NOT_NULL(stub);
    // The last one went out: the oversized one was dropped, not retried.
    TEST_ASSERT_EQUAL_STRING("q/last", stub->lastTopic.c_str());
    TEST_ASSERT_EQUAL_UINT32(2, mqtt.getStatistics().publishCount);
    TEST_ASSERT_EQUAL_UINT32(1, mqtt.getStatistics().publishErrors);
    mqtt.loop();
    TEST_ASSERT_EQUAL_UINT32(1, mqtt.getStatistics().publishErrors);  // and not counted again
    mqtt.shutdown();
    HAL::WiFiImpl::setConnectedForTest(false);
}

void test_publish_now_offline_returns_false_and_queues_nothing() {
    MQTTConfig cfg;
    cfg.broker = "test.local";
    MQTTComponent mqtt(cfg);
    mqtt.begin();
    TEST_ASSERT_FALSE(mqtt.isConnected());

    static const char payload[] = "{\"heap\":1}";
    TEST_ASSERT_FALSE(mqtt.publishNow("dev/telemetry", payload, sizeof(payload) - 1));
    // The mutation this pins: route publishNow through enqueueMessage and the
    // sample sits in the queue, allocated, to be delivered stale.
    TEST_ASSERT_EQUAL_UINT32(0, mqtt.getQueuedMessageCount());
    TEST_ASSERT_EQUAL_UINT32(0, mqtt.getStatistics().publishCount);
    mqtt.shutdown();
}

void test_publish_now_over_the_rate_limit_returns_false_and_queues_nothing() {
    MQTTConfig cfg;
    cfg.broker = "test.local";
    cfg.publishRateLimit = 2;
    cfg.maxQueueSize = 0;
    MQTTComponent mqtt(cfg);
    mqtt.begin();
    HAL::WiFiImpl::setConnectedForTest(true);
    mqtt.connect();
    TEST_ASSERT_TRUE(mqtt.isConnected());

    static const char payload[] = "{\"heap\":1}";
    TEST_ASSERT_TRUE(mqtt.publishNow("dev/telemetry", payload, sizeof(payload) - 1));
    TEST_ASSERT_TRUE(mqtt.publishNow("dev/telemetry", payload, sizeof(payload) - 1));
    TEST_ASSERT_FALSE(mqtt.publishNow("dev/telemetry", payload, sizeof(payload) - 1));
    TEST_ASSERT_EQUAL_UINT32(0, mqtt.getQueuedMessageCount());
    TEST_ASSERT_EQUAL_UINT32(2, mqtt.getStatistics().publishCount);

    mqtt.shutdown();
    HAL::WiFiImpl::setConnectedForTest(false);
}

void test_publish_now_refuses_a_packet_larger_than_the_client_buffer() {
    // PubSubClient answers a bare false; the component says why and counts it.
    MQTTConfig cfg;
    cfg.broker = "test.local";
    MQTTComponent mqtt(cfg);
    mqtt.begin();
    HAL::WiFiImpl::setConnectedForTest(true);
    mqtt.connect();
    std::string big(4000, 'x');
    TEST_ASSERT_FALSE(mqtt.publishNow("dev/telemetry", big.c_str(), big.size()));
    TEST_ASSERT_EQUAL_UINT32(1, mqtt.getStatistics().publishErrors);
    TEST_ASSERT_EQUAL_UINT32(0, mqtt.getStatistics().publishCount);
    mqtt.shutdown();
    HAL::WiFiImpl::setConnectedForTest(false);
}

void test_publish_now_connected_reaches_the_client_and_shares_the_window() {
    // The two counters publish() keeps are kept here too, so a tick at second 0
    // counts against the same window as the application's own publishes.
    MQTTConfig cfg;
    cfg.broker = "test.local";
    cfg.publishRateLimit = 3;
    cfg.maxQueueSize = 0;
    MQTTComponent mqtt(cfg);
    mqtt.begin();
    HAL::WiFiImpl::setConnectedForTest(true);
    mqtt.connect();

    static const char payload[] = "{\"heap\":1}";
    TEST_ASSERT_TRUE(mqtt.publishNow("dev/telemetry", payload, sizeof(payload) - 1, true));
    TEST_ASSERT_EQUAL_UINT32(1, mqtt.getStatistics().publishCount);
    // What the client was handed, not only that it was called: topic, the
    // exact bytes, the retain flag.
    auto* stub = static_cast<HAL::MQTT::MQTTClientImpl*>(mqtt.getClientForTest());
    TEST_ASSERT_NOT_NULL(stub);
    TEST_ASSERT_EQUAL_STRING("dev/telemetry", stub->lastTopic.c_str());
    TEST_ASSERT_EQUAL_STRING("{\"heap\":1}", stub->lastPayload.c_str());
    TEST_ASSERT_TRUE(stub->lastRetained);
    TEST_ASSERT_TRUE(mqtt.publish("topic", "a"));
    TEST_ASSERT_TRUE(mqtt.publish("topic", "b"));
    TEST_ASSERT_TRUE(mqtt.publish("topic", "c"));   // queued: the window is spent
    TEST_ASSERT_EQUAL_UINT32(3, mqtt.getStatistics().publishCount);
    TEST_ASSERT_EQUAL_UINT32(1, mqtt.getQueuedMessageCount());

    mqtt.shutdown();
    HAL::WiFiImpl::setConnectedForTest(false);
}


// ============================================================================
// A link the broker or the network drops
// ============================================================================

namespace {

/// Brings a component up inside a Core, connected to the stub broker.
struct ConnectedBus {
    Core core;
    MQTTComponent* mqtt = nullptr;
    uint32_t connects = 0;
    uint32_t disconnects = 0;
    std::vector<const char*> order;

    explicit ConnectedBus(bool autoReconnect) {
        MQTTConfig cfg;
        cfg.broker = "test.broker.com";
        cfg.port = 1883;
        cfg.enabled = true;
        cfg.autoReconnect = autoReconnect;
        cfg.reconnectDelay = 0;  // Eliminate the timer as a variable
        core.addComponent(std::make_unique<MQTTComponent>(cfg));
        core.begin();
        HAL::WiFiImpl::setConnectedForTest(true);
        mqtt = core.getComponent<MQTTComponent>("MQTT");
        TEST_ASSERT_NOT_NULL_MESSAGE(mqtt, "the fixture has no component to drive");
        core.getEventBus().subscribe(String(MQTTEvents::EVENT_CONNECTED), [this](const void*) {
            connects++;
            order.push_back(MQTTEvents::EVENT_CONNECTED);
        }, nullptr);
        core.getEventBus().subscribe(String(MQTTEvents::EVENT_DISCONNECTED), [this](const void*) {
            disconnects++;
            order.push_back(MQTTEvents::EVENT_DISCONNECTED);
        }, nullptr);
        mqtt->connect();
        core.getEventBus().poll();
    }

    ~ConnectedBus() {
        mqtt->shutdown();
        HAL::WiFiImpl::setConnectedForTest(false);
    }

    /// The broker or the network drops the link: the client goes down under the
    /// component, which is never told.
    void dropTheLink() {
        auto* client = mqtt->getClientForTest();
        TEST_ASSERT_NOT_NULL_MESSAGE(client, "nothing to drop: the fixture never built a client");
        client->disconnect();
    }

    void loopAndPoll(int times = 1) {
        for (int i = 0; i < times; i++) mqtt->loop();
        core.getEventBus().poll();
    }
};

}  // namespace

void test_a_dropped_link_emits_disconnected_once() {
    ConnectedBus bus(/*autoReconnect=*/false);
    TEST_ASSERT_TRUE(bus.mqtt->isConnected());
    TEST_ASSERT_EQUAL_UINT32(1, bus.connects);
    TEST_ASSERT_EQUAL_UINT32(0, bus.disconnects);

    bus.dropTheLink();
    bus.loopAndPoll();

    TEST_ASSERT_EQUAL_UINT32(1, bus.disconnects);
    TEST_ASSERT_FALSE(bus.mqtt->isConnected());
}

void test_a_dropped_link_does_not_repeat_the_event_on_later_loops() {
    ConnectedBus bus(/*autoReconnect=*/false);
    bus.dropTheLink();
    bus.loopAndPoll();
    TEST_ASSERT_EQUAL_UINT32(1, bus.disconnects);

    bus.loopAndPoll(10);

    TEST_ASSERT_EQUAL_UINT32(1, bus.disconnects);
}

void test_a_dropped_link_reports_the_state_it_is_in() {
    // The WebUI reads getState(); over a dead link it said Connected.
    ConnectedBus bus(/*autoReconnect=*/false);
    bus.dropTheLink();
    bus.loopAndPoll();

    TEST_ASSERT_EQUAL(static_cast<int>(MQTTState::Disconnected),
                      static_cast<int>(bus.mqtt->getState()));
}

void test_a_reconnection_announces_the_loss_before_its_own_success() {
    ConnectedBus bus(/*autoReconnect=*/true);
    bus.dropTheLink();
    bus.loopAndPoll();

    // One loop() both notices the loss and reconnects, so the order the bus
    // dispatches them in is the whole assertion.
    TEST_ASSERT_EQUAL_UINT32(1, bus.disconnects);
    TEST_ASSERT_EQUAL_UINT32(2, bus.connects);
    TEST_ASSERT_TRUE(bus.mqtt->isConnected());
    TEST_ASSERT_EQUAL_size_t(3, bus.order.size());
    TEST_ASSERT_EQUAL_STRING(MQTTEvents::EVENT_CONNECTED, bus.order[0]);
    TEST_ASSERT_EQUAL_STRING(MQTTEvents::EVENT_DISCONNECTED, bus.order[1]);
    TEST_ASSERT_EQUAL_STRING(MQTTEvents::EVENT_CONNECTED, bus.order[2]);
}

void test_clearing_the_broker_while_connected_announces_the_loss() {
    // The WebUI writes this field straight through, empty value included, and
    // loop() returns on an empty broker before it can notice anything.
    ConnectedBus bus(/*autoReconnect=*/false);
    bus.mqtt->setBroker("", 0);
    bus.loopAndPoll(3);

    TEST_ASSERT_EQUAL_UINT32(1, bus.disconnects);
    TEST_ASSERT_FALSE(bus.mqtt->isConnected());
    TEST_ASSERT_EQUAL(static_cast<int>(MQTTState::Disconnected),
                      static_cast<int>(bus.mqtt->getState()));
}

void test_a_deliberate_disconnect_still_emits_exactly_once() {
    // disconnect() emits, then loop() sees a client that is down: the pair must
    // not announce the same loss twice.
    ConnectedBus bus(/*autoReconnect=*/false);
    bus.mqtt->disconnect();
    bus.loopAndPoll(3);

    TEST_ASSERT_EQUAL_UINT32(1, bus.disconnects);
}

int main() {
    UNITY_BEGIN();

    // Config limit enforcement tests
    RUN_TEST(test_mqtt_queue_rejects_when_full);
    RUN_TEST(test_mqtt_queue_unlimited_when_zero);
    RUN_TEST(test_mqtt_subscribe_rejects_at_limit);
    RUN_TEST(test_mqtt_subscribe_unlimited_when_zero);
    RUN_TEST(test_mqtt_rate_limit_enforced);
    RUN_TEST(test_mqtt_rate_limit_unlimited_when_zero);

    // Rate-limited messages are deferred, not discarded
    RUN_TEST(test_mqtt_rate_limited_messages_drain_next_window);

    // publishNow
    RUN_TEST(test_publish_now_offline_returns_false_and_queues_nothing);
    RUN_TEST(test_publish_now_over_the_rate_limit_returns_false_and_queues_nothing);
    RUN_TEST(test_publish_now_connected_reaches_the_client_and_shares_the_window);
    RUN_TEST(test_publish_now_refuses_a_packet_larger_than_the_client_buffer);
    RUN_TEST(test_an_oversized_queued_message_is_dropped_and_the_queue_keeps_draining);
    RUN_TEST(test_mqtt_rate_limited_queue_still_bounded);

    // A link the broker or the network drops
    RUN_TEST(test_a_dropped_link_emits_disconnected_once);
    RUN_TEST(test_a_dropped_link_does_not_repeat_the_event_on_later_loops);
    RUN_TEST(test_a_dropped_link_reports_the_state_it_is_in);
    RUN_TEST(test_a_reconnection_announces_the_loss_before_its_own_success);
    RUN_TEST(test_clearing_the_broker_while_connected_announces_the_loss);
    RUN_TEST(test_a_deliberate_disconnect_still_emits_exactly_once);

    return UNITY_END();
}
