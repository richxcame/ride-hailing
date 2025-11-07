package notifications

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FirebaseClient handles Firebase Cloud Messaging operations
type FirebaseClient struct {
	client *messaging.Client
}

// NewFirebaseClient creates a new Firebase client
func NewFirebaseClient(credentialsPath string) (*FirebaseClient, error) {
	ctx := context.Background()

	var opt option.ClientOption
	if credentialsPath != "" {
		opt = option.WithCredentialsFile(credentialsPath)
	} else {
		// Use default credentials from environment
		opt = option.WithCredentialsJSON([]byte{})
	}

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create messaging client: %w", err)
	}

	return &FirebaseClient{client: client}, nil
}

// SendPushNotification sends a push notification to a device
func (f *FirebaseClient) SendPushNotification(ctx context.Context, token, title, body string, data map[string]string) (string, error) {
	message := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				Sound: "default",
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound: "default",
				},
			},
		},
	}

	response, err := f.client.Send(ctx, message)
	if err != nil {
		return "", fmt.Errorf("failed to send push notification: %w", err)
	}

	return response, nil
}

// SendMulticastNotification sends notification to multiple devices
func (f *FirebaseClient) SendMulticastNotification(ctx context.Context, tokens []string, title, body string, data map[string]string) (*messaging.BatchResponse, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no tokens provided")
	}

	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:   data,
		Tokens: tokens,
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
	}

	response, err := f.client.SendEachForMulticast(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to send multicast notification: %w", err)
	}

	return response, nil
}

// SendTopicNotification sends notification to a topic
func (f *FirebaseClient) SendTopicNotification(ctx context.Context, topic, title, body string, data map[string]string) (string, error) {
	message := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	response, err := f.client.Send(ctx, message)
	if err != nil {
		return "", fmt.Errorf("failed to send topic notification: %w", err)
	}

	return response, nil
}

// SubscribeToTopic subscribes tokens to a topic
func (f *FirebaseClient) SubscribeToTopic(ctx context.Context, tokens []string, topic string) error {
	_, err := f.client.SubscribeToTopic(ctx, tokens, topic)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	return nil
}

// UnsubscribeFromTopic unsubscribes tokens from a topic
func (f *FirebaseClient) UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) error {
	_, err := f.client.UnsubscribeFromTopic(ctx, tokens, topic)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe from topic: %w", err)
	}

	return nil
}
