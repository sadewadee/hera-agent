package gateway

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAgentBus_SubscribeAndPublish(t *testing.T) {
	bus := NewAgentBus()
	ch1 := bus.Subscribe("delegation", "agent-a")
	ch2 := bus.Subscribe("delegation", "agent-b")

	msg := AgentMessage{
		Topic:   "delegation",
		From:    "main",
		Payload: "write the auth module",
	}
	bus.Publish(msg)

	// Both subscribers should receive the message.
	select {
	case got := <-ch1:
		assert.Equal(t, "write the auth module", got.Payload)
	case <-time.After(time.Second):
		t.Fatal("agent-a did not receive message within timeout")
	}

	select {
	case got := <-ch2:
		assert.Equal(t, "write the auth module", got.Payload)
	case <-time.After(time.Second):
		t.Fatal("agent-b did not receive message within timeout")
	}

	bus.Unsubscribe("delegation", "agent-a")
	bus.Unsubscribe("delegation", "agent-b")
}

func TestAgentBus_DirectedMessage_OnlyTargetReceives(t *testing.T) {
	bus := NewAgentBus()
	chA := bus.Subscribe("delegation", "agent-a")
	chB := bus.Subscribe("delegation", "agent-b")
	defer bus.Unsubscribe("delegation", "agent-a")
	defer bus.Unsubscribe("delegation", "agent-b")

	msg := AgentMessage{
		Topic:   "delegation",
		From:    "main",
		To:      "agent-b",
		Payload: "only for b",
	}
	bus.Publish(msg)

	// agent-b must receive.
	select {
	case got := <-chB:
		assert.Equal(t, "only for b", got.Payload)
	case <-time.After(time.Second):
		t.Fatal("agent-b did not receive directed message")
	}

	// agent-a must NOT receive.
	select {
	case m := <-chA:
		t.Fatalf("agent-a unexpectedly received: %+v", m)
	case <-time.After(50 * time.Millisecond):
		// Good — no message for agent-a.
	}
}

func TestAgentBus_Unsubscribe_ChannelClosed(t *testing.T) {
	bus := NewAgentBus()
	ch := bus.Subscribe("test", "agent-x")
	bus.Unsubscribe("test", "agent-x")

	// Channel must be closed after Unsubscribe.
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel should be closed after Unsubscribe")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("channel was not closed by Unsubscribe")
	}
}

func TestAgentBus_PublishToEmptyTopic(t *testing.T) {
	bus := NewAgentBus()
	// Publishing with no subscribers should not panic.
	assert.NotPanics(t, func() {
		bus.Publish(AgentMessage{Topic: "empty-topic", Payload: "nobody home"})
	})
}

func TestAgentBus_TopicsAndSubscriberCount(t *testing.T) {
	bus := NewAgentBus()
	bus.Subscribe("alpha", "a1")
	bus.Subscribe("alpha", "a2")
	bus.Subscribe("beta", "b1")
	defer bus.Unsubscribe("alpha", "a1")
	defer bus.Unsubscribe("alpha", "a2")
	defer bus.Unsubscribe("beta", "b1")

	assert.Equal(t, 2, bus.SubscriberCount("alpha"))
	assert.Equal(t, 1, bus.SubscriberCount("beta"))
	assert.Equal(t, 0, bus.SubscriberCount("gamma"))

	topics := bus.Topics()
	assert.Contains(t, topics, "alpha")
	assert.Contains(t, topics, "beta")
}

func TestAgentBus_ConcurrentPublishSubscribe(t *testing.T) {
	bus := NewAgentBus()
	ch := bus.Subscribe("concurrent", "listener")
	defer bus.Unsubscribe("concurrent", "listener")

	const n = 20
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(AgentMessage{Topic: "concurrent", Payload: "x"})
		}()
	}
	wg.Wait()

	// Drain channel — we should see at most n messages (some may have been
	// dropped if the 16-buffer was full, which is expected behaviour).
	received := 0
	deadline := time.After(200 * time.Millisecond)
drain:
	for {
		select {
		case <-ch:
			received++
		case <-deadline:
			break drain
		}
	}
	assert.Greater(t, received, 0, "at least some messages should arrive")
}

// TestAgentBus_DelegateIntegration verifies the delegation posting pattern:
// when DelegateTo is called, the caller can post an observation message on
// the "delegation" topic and all observers receive it.
func TestAgentBus_DelegateIntegration(t *testing.T) {
	bus := NewAgentBus()
	observer := bus.Subscribe("delegation", "logger")
	defer bus.Unsubscribe("delegation", "logger")

	// Simulate what the delegate_task tool does: post on "delegation" topic.
	bus.Publish(AgentMessage{
		Topic:   "delegation",
		From:    "main-agent",
		To:      "",
		Payload: "delegated: write auth module",
	})

	select {
	case msg := <-observer:
		assert.Equal(t, "main-agent", msg.From)
		assert.Contains(t, msg.Payload, "delegated")
	case <-time.After(time.Second):
		t.Fatal("observer did not receive delegation message")
	}
}
