package gateway

import (
	"sync"
)

// AgentMessage is a typed message sent between agents over the AgentBus.
type AgentMessage struct {
	// Topic is the routing key. Subscribers listening on this topic receive
	// the message. Common topics: "delegation", "broadcast", agent-specific names.
	Topic string `json:"topic"`
	// From is the name of the publishing agent.
	From string `json:"from"`
	// To is an optional target agent name. Empty means all subscribers of Topic.
	To string `json:"to,omitempty"`
	// Payload is the free-form content of the message.
	Payload string `json:"payload"`
}

// agentSubscription is a single subscriber on the bus.
type agentSubscription struct {
	id string // subscriber identity (agent name or unique ID)
	ch chan AgentMessage
}

// AgentBus is a pub/sub bus for agent-to-agent messaging.
// Messages are routed by topic. Multiple subscribers may share a topic
// (fan-out). Non-blocking send: if a subscriber's buffer is full, the
// message is dropped for that subscriber (same pattern as mcp.EventBus).
//
// Thread-safe: all methods may be called concurrently.
type AgentBus struct {
	mu   sync.Mutex
	subs map[string][]agentSubscription // keyed by topic
}

// NewAgentBus creates a new, empty AgentBus.
func NewAgentBus() *AgentBus {
	return &AgentBus{subs: make(map[string][]agentSubscription)}
}

// Subscribe registers id as a subscriber on topic and returns a buffered
// receive-only channel. The caller must call Unsubscribe when done.
//
// Buffer size is 16 (same as mcp.EventBus) to tolerate brief back-pressure
// without blocking publishers.
func (b *AgentBus) Subscribe(topic, id string) <-chan AgentMessage {
	ch := make(chan AgentMessage, 16)
	b.mu.Lock()
	b.subs[topic] = append(b.subs[topic], agentSubscription{id: id, ch: ch})
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes the subscription identified by (topic, id) and closes
// the associated channel. If multiple subscriptions share the same id under
// the same topic, only the first match is removed.
func (b *AgentBus) Unsubscribe(topic, id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.subs[topic]
	for i, s := range subs {
		if s.id == id {
			b.subs[topic] = append(subs[:i], subs[i+1:]...)
			close(s.ch)
			return
		}
	}
}

// Publish delivers msg to all subscribers of msg.Topic. If msg.To is
// non-empty, only the subscriber whose id matches msg.To receives it.
// Delivery is non-blocking: a full subscriber buffer causes a silent drop.
func (b *AgentBus) Publish(msg AgentMessage) {
	b.mu.Lock()
	snapshot := make([]agentSubscription, len(b.subs[msg.Topic]))
	copy(snapshot, b.subs[msg.Topic])
	b.mu.Unlock()

	for _, s := range snapshot {
		if msg.To != "" && s.id != msg.To {
			continue
		}
		select {
		case s.ch <- msg:
		default:
			// Subscriber buffer full; drop for this subscriber.
		}
	}
}

// Topics returns a snapshot of all topics that have at least one subscriber.
func (b *AgentBus) Topics() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, 0, len(b.subs))
	for topic, subs := range b.subs {
		if len(subs) > 0 {
			out = append(out, topic)
		}
	}
	return out
}

// SubscriberCount returns the number of subscribers on topic.
func (b *AgentBus) SubscriberCount(topic string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs[topic])
}

// PublishDelegation satisfies the builtin.DelegationObserver interface.
// It posts an AgentMessage on the "delegation" topic so observers can audit
// which agent delegated to which and with what task.
func (b *AgentBus) PublishDelegation(callerAgent, targetAgent, payload string) {
	b.Publish(AgentMessage{
		Topic:   "delegation",
		From:    callerAgent,
		To:      "",
		Payload: payload + " -> " + targetAgent,
	})
}
