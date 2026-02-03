package events

import (
	"sync"
)

// ============================================================================
// EVENT BUS INTERFACE
// ============================================================================

// EventBus defines the interface for publishing and subscribing to domain events
type EventBus interface {
	// Publish publishes an event to all subscribers
	Publish(event DomainEvent)

	// Subscribe registers a handler for a specific event type
	Subscribe(eventType string, handler EventHandler)

	// Unsubscribe removes a handler for a specific event type
	Unsubscribe(eventType string, handler EventHandler)
}

// EventHandler is a function that handles domain events
type EventHandler func(event DomainEvent)

// ============================================================================
// IN-MEMORY EVENT BUS IMPLEMENTATION
// ============================================================================

// InMemoryEventBus is a simple in-memory implementation of EventBus
// This is suitable for monolith deployment and testing
// For distributed systems, replace with Kafka/RabbitMQ/SQS
type InMemoryEventBus struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[string][]EventHandler),
	}
}

// Publish publishes an event to all registered handlers
func (b *InMemoryEventBus) Publish(event DomainEvent) {
	if event == nil {
		return
	}

	eventType := event.EventType()

	b.mu.RLock()
	handlers, exists := b.handlers[eventType]
	b.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		return
	}

	// Execute handlers asynchronously
	// In production, you might want to add:
	// - Error handling
	// - Retry logic
	// - Dead letter queue
	// - Circuit breaker
	for _, handler := range handlers {
		go func(h EventHandler, evt DomainEvent) {
			defer func() {
				if r := recover(); r != nil {
					// Log panic but don't crash
					// TODO: Add proper logging
				}
			}()
			h(evt)
		}(handler, event)
	}
}

// Subscribe registers a handler for a specific event type
func (b *InMemoryEventBus) Subscribe(eventType string, handler EventHandler) {
	if handler == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.handlers[eventType]; !exists {
		b.handlers[eventType] = make([]EventHandler, 0)
	}

	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Unsubscribe removes a handler for a specific event type
func (b *InMemoryEventBus) Unsubscribe(eventType string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	handlers, exists := b.handlers[eventType]
	if !exists {
		return
	}

	// Remove the handler from the slice
	for i, h := range handlers {
		// Compare function pointers (this is a limitation of Go)
		// In production, consider using handler IDs instead
		if &h == &handler {
			b.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// ============================================================================
// SYNCHRONOUS EVENT BUS (for testing)
// ============================================================================

// SyncEventBus is a synchronous event bus for testing
// Events are processed immediately in the same goroutine
type SyncEventBus struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex
}

// NewSyncEventBus creates a new synchronous event bus
func NewSyncEventBus() *SyncEventBus {
	return &SyncEventBus{
		handlers: make(map[string][]EventHandler),
	}
}

// Publish publishes an event synchronously
func (b *SyncEventBus) Publish(event DomainEvent) {
	if event == nil {
		return
	}

	eventType := event.EventType()

	b.mu.RLock()
	handlers, exists := b.handlers[eventType]
	b.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		return
	}

	// Execute handlers synchronously (useful for testing)
	for _, handler := range handlers {
		handler(event)
	}
}

// Subscribe registers a handler for a specific event type
func (b *SyncEventBus) Subscribe(eventType string, handler EventHandler) {
	if handler == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.handlers[eventType]; !exists {
		b.handlers[eventType] = make([]EventHandler, 0)
	}

	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Unsubscribe removes a handler for a specific event type
func (b *SyncEventBus) Unsubscribe(eventType string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	handlers, exists := b.handlers[eventType]
	if !exists {
		return
	}

	for i, h := range handlers {
		if &h == &handler {
			b.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// ============================================================================
// NO-OP EVENT BUS (for testing/development)
// ============================================================================

// NoOpEventBus is a no-op event bus that does nothing
// Useful for testing when you don't want events to be processed
type NoOpEventBus struct{}

// NewNoOpEventBus creates a new no-op event bus
func NewNoOpEventBus() *NoOpEventBus {
	return &NoOpEventBus{}
}

// Publish does nothing
func (b *NoOpEventBus) Publish(event DomainEvent) {
	// Do nothing
}

// Subscribe does nothing
func (b *NoOpEventBus) Subscribe(eventType string, handler EventHandler) {
	// Do nothing
}

// Unsubscribe does nothing
func (b *NoOpEventBus) Unsubscribe(eventType string, handler EventHandler) {
	// Do nothing
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// PublishAll publishes multiple events
func PublishAll(bus EventBus, events ...DomainEvent) {
	for _, event := range events {
		if event != nil {
			bus.Publish(event)
		}
	}
}
