package realtime

import (
	"context"
	"encoding/json"
	"fmt"

	"turtle/infra"
)

type PubSubService struct{}

func NewPubSubService() *PubSubService {
	return &PubSubService{}
}

// Event types
const (
	EventOrderCreated    = "order.created"
	EventOrderUpdated    = "order.updated"
	EventLocationUpdated = "location.updated"
	EventMessageReceived = "message.received"
	EventCaptainAssigned = "captain.assigned"
	EventTypingIndicator = "typing.indicator"
)

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// PublishOrderUpdate publishes order update event
func (s *PubSubService) PublishOrderUpdate(orderID uint, status string, data interface{}) error {
	event := Event{
		Type: EventOrderUpdated,
		Payload: map[string]interface{}{
			"order_id": orderID,
			"status":   status,
			"data":     data,
		},
	}

	return s.publish(fmt.Sprintf("order:%d", orderID), event)
}

// PublishLocationUpdate publishes location update
func (s *PubSubService) PublishLocationUpdate(orderID, captainID uint, lat, lng float64) error {
	event := Event{
		Type: EventLocationUpdated,
		Payload: map[string]interface{}{
			"order_id":   orderID,
			"captain_id": captainID,
			"latitude":   lat,
			"longitude":  lng,
		},
	}

	return s.publish(fmt.Sprintf("order:%d:location", orderID), event)
}

// PublishChatMessage publishes chat message
func (s *PubSubService) PublishChatMessage(roomID, senderID, receiverID uint, message string) error {
	event := Event{
		Type: EventMessageReceived,
		Payload: map[string]interface{}{
			"room_id":     roomID,
			"sender_id":   senderID,
			"receiver_id": receiverID,
			"message":     message,
		},
	}

	return s.publish(fmt.Sprintf("chat:%d", roomID), event)
}

// PublishTypingIndicator publishes typing indicator
func (s *PubSubService) PublishTypingIndicator(roomID, userID uint, isTyping bool) error {
	event := Event{
		Type: EventTypingIndicator,
		Payload: map[string]interface{}{
			"room_id":   roomID,
			"user_id":   userID,
			"is_typing": isTyping,
		},
	}

	return s.publish(fmt.Sprintf("chat:%d:typing", roomID), event)
}

// Subscribe subscribes to events on a channel
func (s *PubSubService) Subscribe(ctx context.Context, channel string) (<-chan *Event, error) {
	pubsub := infra.Redis.Subscribe(ctx, channel)
	ch := make(chan *Event)

	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-pubsub.Channel():
				var event Event
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					continue
				}
				ch <- &event
			}
		}
	}()

	return ch, nil
}

// publish publishes event to Redis
func (s *PubSubService) publish(channel string, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return infra.Redis.Publish(context.Background(), channel, data).Err()
}
