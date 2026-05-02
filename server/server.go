package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iucario/bili-danmu-go/internal/config"
)

// Run starts the HTTP server and blocks until SIGINT or SIGTERM.
// onShutdown is called before the HTTP server stops accepting new connections,
// allowing in-flight SSE rooms to broadcast a final message.
func Run(cfg *config.Config, handler http.Handler, onShutdown func()) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	// baseCtx is the parent of every request context. Cancelling it causes all
	// active SSE handlers (which select on r.Context().Done()) to return, so
	// srv.Shutdown can complete without waiting for clients to disconnect.
	baseCtx, cancelBase := context.WithCancel(context.Background())
	defer cancelBase()

	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		BaseContext:  func(_ net.Listener) context.Context { return baseCtx },
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // disabled — SSE connections are long-lived
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", "http://"+addr)
		fmt.Printf("\n  danmu-go running → http://%s\n\n", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	// os.Interrupt = Ctrl-C on all platforms; SIGTERM for process managers on Unix.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if isAddrInUse(err) {
			return fmt.Errorf("port %d is already in use — change 'port' in data/config.ini", cfg.Port)
		}
		return err
	case sig := <-quit:
		slog.Info("shutting down", "signal", sig)
	}

	if onShutdown != nil {
		onShutdown()
	}

	// Cancel base context so all SSE handlers exit before Shutdown waits.
	cancelBase()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

func isAddrInUse(err error) bool {
	var syscallErr *os.SyscallError
	if errors.As(err, &syscallErr) {
		return errors.Is(syscallErr.Err, syscall.EADDRINUSE)
	}
	return false
}
