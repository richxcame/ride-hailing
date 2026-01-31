package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// Subjects for ride-hailing events.
const (
	SubjectRideRequested = "rides.requested"
	SubjectRideAccepted  = "rides.accepted"
	SubjectRideStarted   = "rides.started"
	SubjectRideCompleted = "rides.completed"
	SubjectRideCancelled = "rides.cancelled"

	SubjectPaymentProcessed = "payments.processed"
	SubjectPaymentFailed    = "payments.failed"

	SubjectDriverLocationUpdated = "drivers.location.updated"
	SubjectDriverOnline          = "drivers.online"
	SubjectDriverOffline         = "drivers.offline"

	SubjectFraudDetected = "fraud.detected"
)

// Event is the envelope for all events published through the bus.
type Event struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// NewEvent creates a new event with a unique ID and current timestamp.
func NewEvent(eventType, source string, data interface{}) (*Event, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal event data: %w", err)
	}
	return &Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().UTC(),
		Data:      raw,
	}, nil
}

// HandlerFunc processes a received event. Return nil to ack, error to nack.
type HandlerFunc func(ctx context.Context, event *Event) error

// Config holds NATS connection settings.
type Config struct {
	URL       string
	Name      string // client connection name
	StreamName string // JetStream stream name (default: "RIDEHAILING")
}

// DefaultConfig returns sensible defaults for local development.
func DefaultConfig() Config {
	return Config{
		URL:        nats.DefaultURL,
		Name:       "ride-hailing",
		StreamName: "RIDEHAILING",
	}
}

// Bus wraps a NATS JetStream connection for publishing and subscribing.
type Bus struct {
	conn   *nats.Conn
	js     jetstream.JetStream
	cfg    Config
	subs   []jetstream.ConsumeContext
}

// New connects to NATS and ensures the JetStream stream exists.
func New(cfg Config) (*Bus, error) {
	nc, err := nats.Connect(cfg.URL,
		nats.Name(cfg.Name),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				logger.Warn("NATS disconnected", zap.Error(err))
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			logger.Info("NATS reconnected")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream init: %w", err)
	}

	// Create or update the stream
	streamName := cfg.StreamName
	if streamName == "" {
		streamName = "RIDEHAILING"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{"rides.>", "payments.>", "drivers.>", "fraud.>"},
		Storage:   jetstream.FileStorage,
		Retention: jetstream.InterestPolicy,
		MaxAge:    72 * time.Hour,
		Replicas:  1,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create stream: %w", err)
	}

	logger.Info("NATS event bus connected",
		zap.String("url", cfg.URL),
		zap.String("stream", streamName),
	)

	return &Bus{conn: nc, js: js, cfg: cfg}, nil
}

// Publish sends an event to the given subject with JetStream guarantees.
func (b *Bus) Publish(ctx context.Context, subject string, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	_, err = b.js.Publish(ctx, subject, data,
		jetstream.WithMsgID(event.ID),
	)
	if err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}

	logger.Debug("event published",
		zap.String("subject", subject),
		zap.String("event_id", event.ID),
		zap.String("type", event.Type),
	)
	return nil
}

// Subscribe creates a durable consumer and processes messages with the handler.
// The consumerName should be unique per subscribing service (e.g., "notifications-rides").
func (b *Bus) Subscribe(ctx context.Context, subject, consumerName string, handler HandlerFunc) error {
	streamName := b.cfg.StreamName
	if streamName == "" {
		streamName = "RIDEHAILING"
	}

	consumer, err := b.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    5,
		DeliverPolicy: jetstream.DeliverNewPolicy,
	})
	if err != nil {
		return fmt.Errorf("create consumer %s: %w", consumerName, err)
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			logger.Warn("failed to unmarshal event", zap.Error(err))
			msg.Term() // don't redeliver malformed messages
			return
		}

		if err := handler(ctx, &event); err != nil {
			logger.Warn("event handler error, will retry",
				zap.String("event_id", event.ID),
				zap.String("type", event.Type),
				zap.Error(err),
			)
			msg.Nak() // redeliver
			return
		}

		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("consume %s: %w", consumerName, err)
	}

	b.subs = append(b.subs, cc)
	logger.Info("subscribed to events",
		zap.String("subject", subject),
		zap.String("consumer", consumerName),
	)
	return nil
}

// SubscribeAll subscribes to a wildcard subject (e.g., "rides.>").
func (b *Bus) SubscribeAll(ctx context.Context, subjectPattern, consumerName string, handler HandlerFunc) error {
	return b.Subscribe(ctx, subjectPattern, consumerName, handler)
}

// Close drains subscriptions and closes the NATS connection.
func (b *Bus) Close() {
	for _, sub := range b.subs {
		sub.Stop()
	}
	if b.conn != nil {
		b.conn.Drain()
	}
	logger.Info("NATS event bus closed")
}

// Connected returns true if the NATS connection is active.
func (b *Bus) Connected() bool {
	return b.conn != nil && b.conn.IsConnected()
}
