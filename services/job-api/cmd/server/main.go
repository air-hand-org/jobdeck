// Command server runs the job API gRPC server.
package main

// ref: https://github.com/connectrpc/connect-go/blob/main/internal/example/server/main.go
// ref: https://github.com/grpc/grpc-go/blob/master/examples/features/health/server/main.go
import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/auth"
	"jobdeck/services/job-api/internal/domain"
	"jobdeck/services/job-api/internal/eventing"
	job_pubsub "jobdeck/services/job-api/internal/eventing/pubsub"
	"jobdeck/services/job-api/internal/repository"
	"jobdeck/services/job-api/internal/telemetry"
	"jobdeck/services/job-api/internal/transport"
	"jobdeck/services/job-api/internal/transport/interceptor"

	jobv1 "github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1"

	"buf.build/go/protovalidate"
	"cloud.google.com/go/pubsub/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	protovalidate_middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

func runHealthCheck(ctx context.Context, h *health.Server, pool *pgxpool.Pool) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	check := func() {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if err := pool.Ping(pingCtx); err != nil {
			h.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
			return
		}

		h.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
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

	shutdownTelemetry, err := telemetry.Init(signalCtx, "job-api")
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

	connStr := os.Getenv("POSTGRESDB_URI")
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatalf("failed to parse conn string %v", err)
	}
	cfg.ConnConfig.Tracer = otelpgx.NewTracer()
	pool, err := pgxpool.NewWithConfig(signalCtx, cfg)
	if err != nil {
		log.Fatalf("failed to connect postgres %v", err)
	}
	defer pool.Close()

	store := repository.NewStore(pool)

	validator, err := protovalidate.New()
	if err != nil {
		log.Fatalf("failed to new proto validate")
	}

	verifier, err := auth.NewJWTVerifier(os.Getenv("AUTH_JWT_PUBLIC_KEY"))
	if err != nil {
		log.Fatalf("failed to new jwt verifier: %v", err)
	}

	authenticator := app.NewTokenAuthenticatorService(store, verifier)

	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			interceptor.ErrorMappingUnaryServerInterceptor(),
			interceptor.RejectNilRequestUnaryServerInterceptor(),
			interceptor.AuthenticationInterceptor(
				selector.MatchFunc(func(_ context.Context, callMeta interceptors.CallMeta) bool {
					return callMeta.FullMethod() != "/grpc.health.v1.Health/Check"
				}),
				func(ctx context.Context, rawToken string) (domain.Actor, error) {
					return authenticator.Authenticate(ctx, rawToken)
				},
			),
			protovalidate_middleware.UnaryServerInterceptor(validator),
			interceptor.LoggingErrorUnaryServerInterceptor(slog.Default()),
		),
	)

	pubsubClient, err := pubsub.NewClient(signalCtx, os.Getenv("PUBSUB_PROJECT_ID"))
	if err != nil {
		log.Fatalf("failed to new client: %v", err)
	}
	defer func() { _ = pubsubClient.Close() }()

	executionPublisher := job_pubsub.NewExecutionPublisher(pubsubClient, os.Getenv("PUBSUB_WORKER_TOPIC_ID"))
	defer executionPublisher.Close()

	outboxDispatcher := eventing.NewOutboxDispatcher(executionPublisher, store)

	g, ctx := errgroup.WithContext(signalCtx)

	healthcheck := health.NewServer()
	healthpb.RegisterHealthServer(s, healthcheck)
	jobv1.RegisterJobServiceServer(s, transport.NewJobServiceServer(app.NewJobUsecase(store)))
	jobv1.RegisterExecutionQueueServiceServer(s, transport.NewExecutionQueueServiceServer(app.NewExecutionQueueUsecase(store, outboxDispatcher)))
	jobv1.RegisterExecutionServiceServer(s, transport.NewExecutionServiceServer(app.NewExecutionUsecase(store, outboxDispatcher)))
	if strings.EqualFold(os.Getenv("GRPC_REFLECTION_ENABLED"), "true") {
		reflection.Register(s)
	}

	g.Go(func() error {
		runHealthCheck(ctx, healthcheck, pool)
		return nil
	})

	g.Go(func() error {
		return outboxDispatcher.Run(ctx)
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
