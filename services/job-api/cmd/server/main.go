package main

// ref: https://github.com/connectrpc/connect-go/blob/main/internal/example/server/main.go
// ref: https://github.com/grpc/grpc-go/blob/master/examples/features/health/server/main.go
import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	jobv1 "jobdeck/gen/jobdeck/job/v1"
	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/repository"
	"jobdeck/services/job-api/internal/transport"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	sleep  = time.Second * 5
	system = "" // empty string represents the health of the system
)

func main() {
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9090"
	}
	connStr := os.Getenv("POSTGRESDB_URI")
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("failed to connect postgres %v", err)
	}
	defer pool.Close()

	store := repository.NewStore(pool)

	s := grpc.NewServer()
	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(s, healthcheck)
	jobv1.RegisterJobServiceServer(s, transport.NewJobServiceServer(app.NewJobUsecase(store)))
	jobv1.RegisterExecutionQueueServiceServer(s, transport.NewExecutionQueueServiceServer(app.NewExecutionUsecase(store)))
	reflection.Register(s)

	// FIXME
	go func() {
		next := healthpb.HealthCheckResponse_SERVING

		for {
			healthcheck.SetServingStatus(system, next)

			if next == healthpb.HealthCheckResponse_SERVING {
				next = healthpb.HealthCheckResponse_NOT_SERVING
			} else {
				next = healthpb.HealthCheckResponse_SERVING
			}

			time.Sleep(sleep)
		}
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
