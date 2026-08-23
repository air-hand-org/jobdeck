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

// https://github.com/GoogleCloudPlatform/golang-samples/blob/main/pubsub/topics/create.go
func ensureTopic(ctx context.Context, projectID string, topicID string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("pubsub.NewClient: %w", err)
	}
	defer client.Close()

	topicName := fmt.Sprintf("projects/%s/topics/%s", projectID, topicID)

	for {
		_, err := client.TopicAdminClient.GetTopic(ctx, &pubsubpb.GetTopicRequest{
			Topic: topicName,
		})

		code := status.Code(err)
		if code == codes.OK {
			slog.Info("Topic already exists", "name", topicName)
			return nil
		}

		if code == codes.NotFound {
			topic := &pubsubpb.Topic{
				Name: topicName,
			}
			_, err := client.TopicAdminClient.CreateTopic(ctx, topic)
			if err == nil || status.Code(err) == codes.AlreadyExists {
				slog.Info("Topic created", "name", topicName)
				return nil
			}
		}

		if code != codes.Unavailable && code != codes.DeadlineExceeded {
			return fmt.Errorf("Failed %w", err)
		}

		slog.Info("Retrying...")
		time.Sleep(3 * time.Second)
	}
}

func main() {
	signalCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	projectID := os.Getenv("PUBSUB_PROJECT_ID")
	topicID := os.Getenv("PUBSUB_TOPIC_ID")
	if projectID == "" || topicID == "" {
		slog.Error("PUBSUB_PROJECT_ID nor PUBSUB_TOPIC_ID not set", "project", projectID, "topic", topicID)
		os.Exit(1)
	}
	err := ensureTopic(signalCtx, projectID, topicID)
	if err != nil {
		slog.Error("create topic failed", "error", err)
		os.Exit(1)
	}
}
