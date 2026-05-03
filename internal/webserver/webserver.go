// Package webserver implements a web server with optional TLS and
// graceful shutdown.
package webserver

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// Options is used to configure a WebServer.
type Options struct {
	ListenAddress string
	TLSCertPath   string
	TLSKeyPath    string
}

// WebServer represents a web server that serves the Expense Tracker
// application over http.
type WebServer struct {
	server     *http.Server
	handlerMux *http.ServeMux
	title      string
	options    *Options
	tlsEnabled bool
}

// New constructs a new WebServer.
func New(title string, options *Options) *WebServer {
	if options == nil {
		options = &Options{
			ListenAddress: ":8080",
			TLSCertPath:   "",
			TLSKeyPath:    "",
		}
	}

	tlsEnabled := !(options.TLSCertPath == "" || options.TLSKeyPath == "")

	handlerMux := &http.ServeMux{}
	httpserver := &http.Server{
		Addr:    options.ListenAddress,
		Handler: handlerMux,
	}

	return &WebServer{
		server:     httpserver,
		handlerMux: handlerMux,
		title:      title,
		options:    options,
		tlsEnabled: tlsEnabled,
	}
}

func (ws *WebServer) HandlerMux() *http.ServeMux {
	return ws.handlerMux
}

func (ws *WebServer) ListenAndServe() {
	// Use a channel to signal server closure
	serverClosed := make(chan struct{})

	// Handle graceful shutdown
	go func() {
		signalReceived := make(chan os.Signal, 1)

		// Handle SIGINT and SIGTERM
		signal.Notify(signalReceived, os.Interrupt, syscall.SIGTERM)

		// Wait for signal
		<-signalReceived

		log.Println("Server shutting down...")
		if err := ws.server.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Fatalf("Error during HTTP server shutdown: %v.", err)
		}

		close(serverClosed)
	}()

	// Start listening using the server
	if ws.tlsEnabled {
		log.Printf("Server starting on %v (TLS)...\n", ws.options.ListenAddress)
		if err := ws.server.ListenAndServeTLS(ws.options.TLSCertPath, ws.options.TLSKeyPath); err != http.ErrServerClosed {
			log.Fatalf("The server failed with the following error: %v.\n", err)
		}
	} else {
		log.Printf("Server starting on %v...\n", ws.options.ListenAddress)
		if err := ws.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("The server failed with the following error: %v.\n", err)
		}
	}

	// Wait for server close
	<-serverClosed

	log.Println("Server shut down.")
}
