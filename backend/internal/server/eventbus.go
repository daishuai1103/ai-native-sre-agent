package server

import (
	"log"
	"sync"

	"github.com/ai-native-sre-agent/backend/internal/models"
)

// eventBus manages SSE event broadcasting to multiple subscribers.
type eventBus struct {
	subscribers map[string][]chan models.StepEvent // sessionID -> list of subscriber channels
	mu          sync.RWMutex
	bufferSize  int
}

func newEventBus(bufferSize int) *eventBus {
	return &eventBus{
		subscribers: make(map[string][]chan models.StepEvent),
		bufferSize:  bufferSize,
	}
}

// Subscribe adds a new subscriber for a session.
// Returns a channel that receives events for that session.
// Caller must call Unsubscribe when done.
func (b *eventBus) Subscribe(sessionID string) <-chan models.StepEvent {
	ch := make(chan models.StepEvent, b.bufferSize)

	b.mu.Lock()
	b.subscribers[sessionID] = append(b.subscribers[sessionID], ch)
	b.mu.Unlock()

	log.Printf("[SSE] new subscriber for session %s (total: %d)", sessionID, len(b.subscribers[sessionID]))
	return ch
}

// Unsubscribe removes a subscriber channel for a session.
func (b *eventBus) Unsubscribe(sessionID string, ch <-chan models.StepEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs, ok := b.subscribers[sessionID]
	if !ok {
		return
	}

	for i, sub := range subs {
		if sub == ch {
			// Remove the subscriber
			b.subscribers[sessionID] = append(subs[:i], subs[i+1:]...)
			close(sub)
			if len(b.subscribers[sessionID]) == 0 {
				delete(b.subscribers, sessionID)
			}
			log.Printf("[SSE] subscriber removed for session %s (remaining: %d)", sessionID, len(b.subscribers[sessionID]))
			return
		}
	}
}

// Publish sends an event to all subscribers of the event's session.
func (b *eventBus) Publish(event models.StepEvent) {
	b.mu.RLock()
	subs, ok := b.subscribers[event.SessionID]
	b.mu.RUnlock()

	if !ok || len(subs) == 0 {
		return
	}

	for _, sub := range subs {
		select {
		case sub <- event:
			// sent successfully
		default:
			// channel full, drop the event (non-blocking)
			log.Printf("[SSE] event dropped for session %s (channel full)", event.SessionID)
		}
	}
}

// PublishGlobal sends an event to ALL subscribers (e.g. alert_detected).
func (b *eventBus) PublishGlobal(event models.StepEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, subs := range b.subscribers {
		for _, sub := range subs {
			select {
			case sub <- event:
			default:
			}
		}
	}
}

// SubscriberCount returns total number of subscribers.
func (b *eventBus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	count := 0
	for _, subs := range b.subscribers {
		count += len(subs)
	}
	return count
}
