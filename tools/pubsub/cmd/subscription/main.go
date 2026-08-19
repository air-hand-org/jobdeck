package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ensureWithEndpoint(ctx context.Context, projectID, topic, subscription, pushEndpoint string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("pubsub.NewClient: %w", err)
	}
	defer client.Close()

	for {
		_, err := client.SubscriptionAdminClient.GetSubscription(ctx, &pubsubpb.GetSubscriptionRequest{
			Subscription: subscription,
		})

		code := status.Code(err)
		if code == codes.OK {
			slog.Info("subscription already exists", "name", subscription)
			return nil
		}

		if code == codes.NotFound {
			_, err := client.SubscriptionAdminClient.CreateSubscription(ctx, &pubsubpb.Subscription{
				Name: subscription,
				Topic: topic,
				AckDeadlineSeconds: 60,
				PushConfig: &pubsubpb.PushConfig{PushEndpoint: pushEndpoint},
			})
			if err == nil || status.Code(err) == codes.AlreadyExists {
				slog.Info("Subscription created", "name", subscription, "endpoint", pushEndpoint)
				return nil
			}
		}

		if code != codes.Unavailable && code != codes.DeadlineExceeded {
			return fmt.Errorf("Failed: %w", err)
		}

		slog.Info("Retrying...")
		time.Sleep(3*time.Second)
	}
}

func main() {
	signalCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	projectID := os.Getenv("PUBSUB_PROJECT_ID")
	topicID := os.Getenv("PUBSUB_TOPIC_ID")
	subscriptionID := os.Getenv("PUBSUB_SUBSCRIPTION_ID")
	pushEndpoint := os.Getenv("PUBSUB_PUSH_ENDPOINT")
	if projectID == "" || topicID == "" || subscriptionID == "" || pushEndpoint == "" {
		slog.Error("PUBSUB_PROJECT_ID nor PUBSUB_TOPIC_ID nor PUBSUB_SUBSCRIPTION_ID nor PUBSUB_PUSH_ENDPOINT not set", "project", projectID, "topic", topicID, "subscription", subscriptionID, "pushendpoint", pushEndpoint)
		os.Exit(1)
	}

	topic := fmt.Sprintf("projects/%s/topics/%s", projectID, topicID)
	subscription := fmt.Sprintf("projects/%s/subscriptions/%s", projectID, subscriptionID)
	if err := ensureWithEndpoint(signalCtx, projectID, topic, subscription, pushEndpoint); err != nil {
		slog.Error("create push subscription failed", "error", err)
		os.Exit(1)
	}

}
