package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"syscall"

	"github.com/davidmdm/x/xcontext"
)

//go:embed content
var content embed.FS

type KnownPages struct {
	Home          []byte
	NotFound      []byte
	InternalError []byte
}

var knownPages = KnownPages{
	Home:          Must2(content.ReadFile("content/pages/home.html")),
	NotFound:      Must2(content.ReadFile("content/pages/not_found.html")),
	InternalError: Must2(content.ReadFile("content/pages/internal_error.html")),
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if !errors.Is(err, context.Canceled) {
			os.Exit(1)
		}
	}
}

func run() error {
	ctx, cancel := xcontext.WithSignalCancelation(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := GetConfig()

	svr := http.Server{
		Addr:        cfg.Port,
		Handler:     Handler(),
		ReadTimeout: cfg.ReadTimeout,
	}

	e := make(chan error, 1)
	go func() {
		fmt.Println("listening on port", cfg.Port)
		if err := svr.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			e <- err
		}
		close(e)
	}()

	select {
	// Server fails to startup and returns a non Server Closed error.
	case err := <-e:
		return fmt.Errorf("failed to start server: %w", err)

	// We have received a cancelation signal, and want to attempt to shutdown our application as gracefully as possible.
	case <-ctx.Done():
		{
			graceCtx, cancel := context.WithTimeout(context.Background(), cfg.GracePeriod)
			defer cancel()

			if err := svr.Shutdown(graceCtx); err != nil {
				return fmt.Errorf("failed to shutdown server: %w", err)
			}
			return context.Cause(ctx)
		}
	}
}
