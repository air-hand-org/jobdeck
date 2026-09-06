// Package main starts the demo HTTP server.
package main

import (
	"context"
	"fmt"
	"jobdeck/services/demo/internal/transport"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"golang.org/x/sync/errgroup"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	signalCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	g, ctx := errgroup.WithContext(signalCtx)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: transport.NewHTTPHandler(),
	}

	g.Go(func() error {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return errors.Wrap(err, "failed to start HTTP server")
		}
		return nil
	})

	g.Go(func() error {
		<-ctx.Done()
		slog.Info("shutting down server")

		ctxS, cancelS := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelS()

		if err := server.Shutdown(ctxS); err != nil {
			return errors.Wrap(err, "failed to shutdown http server")
		}
		slog.Info("shut down server")
		return nil
	})

	if err := g.Wait(); err != nil {
		slog.Error("error occured", "err", err)
		os.Exit(1)
	}
}
