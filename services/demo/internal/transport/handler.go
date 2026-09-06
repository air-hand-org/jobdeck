// Package transport provides the demo HTTP handlers.
package transport

import (
	"net/http"
	"time"
)

// NewHTTPHandler returns the demo HTTP handler multiplexer.
func NewHTTPHandler() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/healthz", healthzHandler)

	mux.HandleFunc("/200", okHandler)
	mux.HandleFunc("/404", notFoundHandler)
	mux.HandleFunc("/504", gatewayTimeoutHandler)
	mux.HandleFunc("/sleep/60", sleep60secHandler)
	return mux
}

func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func homeHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Welcome Demo App\n"))
}

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func notFoundHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("not found\n"))
}

func gatewayTimeoutHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusGatewayTimeout)
	_, _ = w.Write([]byte("gateway timeout\n"))
}

func sleep60secHandler(w http.ResponseWriter, _ *http.Request) {
	time.Sleep(60 * time.Second)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("sleep 60"))
}
