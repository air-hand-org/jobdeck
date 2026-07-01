package main

// ref: https://github.com/connectrpc/connect-go/blob/main/internal/example/server/main.go
// ref: https://github.com/grpc/grpc-go/blob/master/examples/features/health/server/main.go
import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	jobv1 "jobdeck/gen/jobdeck/job/v1"
	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/repository"
	"jobdeck/services/job-api/internal/transport"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/jackc/pgx/v5/pgxpool"
)

func runHealthCheck(ctx context.Context, h *health.Server, pool *pgxpool.Pool) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	check := func() {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if err := pool.Ping(pingCtx); err != nil {
			h.SetServingStatus("", healthgrpc.HealthCheckResponse_NOT_SERVING)
			return
		}

		h.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	}

	check()

	for {
		select {
		case <-ctx.Done():
			h.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
			return
		case <-ticker.C:
			check()
		}
	}
}

func main() {
	signalCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9090"
	}

	connStr := os.Getenv("POSTGRESDB_URI")
	pool, err := pgxpool.New(signalCtx, connStr)
	if err != nil {
		log.Fatalf("failed to connect postgres %v", err)
	}
	defer pool.Close()

	store := repository.NewStore(pool)

	s := grpc.NewServer()
	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(s, healthcheck)
	jobv1.RegisterJobServiceServer(s, transport.NewJobServiceServer(app.NewJobUsecase(store)))
	jobv1.RegisterExecutionQueueServiceServer(s, transport.NewExecutionQueueServiceServer(app.NewExecutionQueueUsecase(store)))
	jobv1.RegisterExecutionServiceServer(s, transport.NewExecutionServiceServer(app.NewExecutionUsecase(store)))
	reflection.Register(s)

	g, ctx := errgroup.WithContext(signalCtx)

	g.Go(func() error {
		runHealthCheck(ctx, healthcheck, pool)
		return nil
	})

	g.Go(func() error {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
		if err != nil {
			return err
		}
		if err := s.Serve(lis); err != nil {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		log.Fatalf("%+v", err)
	}
}
