// Package healthroutes provides HTTP health check endpoints for use with
// orchestrators such as Kubernetes.
//
// Mount the handler returned by NewHandler on /healthz/:
//
//	mux.Handle("/healthz/", http.StripPrefix("/healthz", healthroutes.NewHandler(store)))
//
// Endpoints:
//
//	GET /live  — always 200; confirms the process is running.
//	GET /ready — calls Checker.Ready; returns 200 or 503.
package healthroutes

import (
	"context"
	"log"
	"net/http"
)

// Checker is implemented by anything that can report whether it is ready
// to serve traffic (e.g. a database store that has applied all migrations).
type Checker interface {
	Ready(ctx context.Context) error
}

// NewHandler returns an HTTP handler for /live and /ready.
// Mount it under /healthz/ using http.StripPrefix.
func NewHandler(c Checker) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /live", handleLive)
	mux.HandleFunc("GET /ready", handleReady(c))
	return mux
}

// handleLive always returns 200. It confirms the process is running.
// No external dependencies are checked.
func handleLive(w http.ResponseWriter, r *http.Request) {
	log.Println("Liveness probe succeeded.")
	w.WriteHeader(http.StatusOK)
}

// handleReady calls Checker.Ready using the request context.
// Returns 200 if ready, 503 if not.
func handleReady(c Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := c.Ready(r.Context()); err != nil {
			log.Printf("Readiness probe failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		log.Println("Readiness probe succeeded.")
		w.WriteHeader(http.StatusOK)
	}
}
