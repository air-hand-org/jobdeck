package main

import (
	"context"
	"flag"
	"jobdeck/services/job-worker/internal/app"
	"jobdeck/services/job-worker/internal/claimer"
	"jobdeck/services/job-worker/internal/telemetry"
	"jobdeck/services/job-worker/internal/worker"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	health_grpc "google.golang.org/grpc/health/grpc_health_v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type config struct {
	workerCount uint
}

func parseFlags() config {
	var cfg config

	flag.UintVar(&cfg.workerCount, "worker", 5, "worker count")
	flag.Parse()

	return cfg
}

func waitForServing(ctx context.Context, conn *grpc.ClientConn) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	client := health_grpc.NewHealthClient(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		rpcCtx, rpcCancel := context.WithTimeout(ctx, 2*time.Second)
		res, err := client.Check(rpcCtx, &healthpb.HealthCheckRequest{
			Service: "",
		})
		rpcCancel()

		if err == nil && res.GetStatus() == health_grpc.HealthCheckResponse_SERVING {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func main() {
	cfg := parseFlags()

	signalCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	shutdownTelemetry, err := telemetry.Init(signalCtx, "job-worker")
	if err != nil {
		log.Fatalf("initialize telemetry: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := shutdownTelemetry(ctx); err != nil {
			log.Printf("shutdown telemetry: %v", err)
		}
	}()

	items := make(chan app.QueuedItem, cfg.workerCount)
	defer close(items)
	results := make(chan app.QueuedResult, cfg.workerCount)
	defer close(results)

	g, ctx := errgroup.WithContext(signalCtx)
	for i := 0; i < int(cfg.workerCount); i++ {
		g.Go(func() error {
			worker := worker.NewWorker()
			err := worker.Run(ctx, i, items, results)
			if err != nil {
				return errors.Wrapf(err, "failed to worker run [%d]", i)
			}
			return nil
		})
	}

	g.Go(func() error {
		server_uri, found := os.LookupEnv("JOB_API_SERVER_URI")
		if !found {
			return errors.New("JOB_API_SERVER_URI is required")
		}
		conn, err := grpc.NewClient(server_uri,
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
			grpc.WithTransportCredentials(
				insecure.NewCredentials(),
			),
			grpc.WithDefaultCallOptions(
				grpc.WaitForReady(true),
			),
		)
		if err != nil {
			return errors.Wrap(err, "failed to new grpc client")
		}
		defer conn.Close()
		conn.Connect()

		if err := waitForServing(ctx, conn); err != nil {
			return errors.Wrap(err, "job-api did not become serving")
		}

		claimer := claimer.NewClaimer(conn, int(cfg.workerCount), 20*time.Second)
		return claimer.Run(ctx, items, results)
	})

	if err := g.Wait(); err != nil {
		log.Fatalf("%+v", err)
	}
}
