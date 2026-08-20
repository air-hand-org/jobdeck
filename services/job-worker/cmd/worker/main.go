package main

import (
	"context"
	"crypto/tls"
	"flag"
	"jobdeck/services/job-worker/internal/app"
	"jobdeck/services/job-worker/internal/auth"
	"jobdeck/services/job-worker/internal/claimer"
	"jobdeck/services/job-worker/internal/reporter"

	"jobdeck/services/job-worker/internal/telemetry"
	"jobdeck/services/job-worker/internal/transport"
	"jobdeck/services/job-worker/internal/transport/interceptor"
	"jobdeck/services/job-worker/internal/worker"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	health_grpc "google.golang.org/grpc/health/grpc_health_v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type config struct {
	workerCount uint
}

func newTLSClient(endpoint string, options ...grpc.DialOption) (*grpc.ClientConn, error) {
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "parse gRPC endpoint")
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return nil, errors.New("gRPC endpoint must be an absolute HTTPS URL")
	}

	options = append(options, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		ServerName: parsed.Hostname(),
	})))
	return grpc.NewClient(parsed.Host, options...)
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
	logger := slog.Default()
	if err := func(logger *slog.Logger) error {
		cfg := parseFlags()

		signalCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		shutdownTelemetry, err := telemetry.Init(signalCtx, "job-worker")
		if err != nil {
			return errors.Wrap(err, "initialize telemetry")
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := shutdownTelemetry(ctx); err != nil {
				logger.Warn(
					"shutdown telemetry",
					"error", err,
				)
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
				if err := worker.Run(ctx, i, items, results); err != nil {
					return errors.Wrapf(err, "failed to worker run [%d]", i)
				}
				return nil
			})
		}

		authEndpoint, found := os.LookupEnv("AUTH_GRPC_ENDPOINT")
		if !found {
			return errors.New("AUTH_GRPC_ENDPOINT is required")
		}
		auth_conn, err := newTLSClient(authEndpoint,
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
			grpc.WithDefaultCallOptions(
				grpc.WaitForReady(true),
			),
		)
		if err != nil {
			return err
		}
		defer auth_conn.Close()
		auth_conn.Connect()
		if err := waitForServing(ctx, auth_conn); err != nil {
			return errors.Wrap(err, "auth did not become serving")
		}

		token_source, err := auth.NewTokenSource(auth_conn, func() time.Time {
			return time.Now()
		}, os.Getenv("AUTH_CLIENT_ID"), os.Getenv("AUTH_CLIENT_SECRET"))
		if err != nil {
			return err
		}

		jobApiEndpoint, found := os.LookupEnv("JOB_API_GRPC_ENDPOINT")
		if !found {
			return errors.New("JOB_API_GRPC_ENDPOINT is required")
		}
		api_conn, err := newTLSClient(jobApiEndpoint,
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
			grpc.WithDefaultCallOptions(
				grpc.WaitForReady(true),
			),
			grpc.WithChainUnaryInterceptor(
				interceptor.AuthUnaryInterceptor(token_source),
			),
		)
		if err != nil {
			return errors.Wrap(err, "failed to new grpc client")
		}
		defer api_conn.Close()
		api_conn.Connect()

		if err := waitForServing(ctx, api_conn); err != nil {
			return errors.Wrap(err, "job-api did not become serving")
		}

		port := os.Getenv("PORT")
		if port == "" {
			port = "9092"
		}
		server := transport.NewServer(port, app.NewPushUsecase(
			claimer.NewClaimer(api_conn, items, logger),
		))

		g.Go(func() error {
			reporter := reporter.NewReporter(api_conn, logger)
			return reporter.Run(ctx, results)
		})

		g.Go(func() error {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				return errors.Wrap(err, "failed to start HTTP server")
			}
			return nil
		})

		g.Go(func() error {
			<-ctx.Done()
			logger.Info("shutting down server")

			ctxS, cancelS := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancelS()

			if err := server.Shutdown(ctxS); err != nil {
				return errors.Wrap(err, "failed to shutdown http server")
			}
			logger.Info("shutdown server")
			return nil
		})

		return g.Wait()
	}(logger); err != nil {
		logger.Error("error occured", "error", err)
		os.Exit(1)
	}
}
