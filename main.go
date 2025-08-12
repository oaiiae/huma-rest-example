package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/danielgtaylor/huma/v2/humacli"

	"github.com/oaiiae/huma-rest-example/handler"
	"github.com/oaiiae/huma-rest-example/router"
	"github.com/oaiiae/huma-rest-example/store"
)

// Options for the CLI. Pass `--port` or set the `SERVICE_PORT` env var.
type Options struct {
	Port int `help:"Port to listen on" short:"p" default:"8888"`
}

func main() {
	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		server := http.Server{
			Addr: fmt.Sprintf(":%d", options.Port),
			Handler: router.New("My API", "1.0.0",
				func(w http.ResponseWriter, r *http.Request) {},
				metrics.WriteProcessMetrics,
				slogWriter{logger.With("tag", "api")},
				(&handler.Greeting{}).Register,
				(&handler.Contacts{Store: new(store.ContactsInmem)}).Register,
			),
			ReadHeaderTimeout: 15 * time.Second,
		}
		hooks.OnStart(func() {
			err := server.ListenAndServe()
			if err != http.ErrServerClosed {
				slog.Error("failed to listen and serve", "err", err)
			} else {
				slog.Info("server closed")
			}
		})
		hooks.OnStop(func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			err := server.Shutdown(ctx)
			if err != nil {
				slog.Warn("could not shutdown the server", "err", err)
			}
		})
	})
	cli.Run()
}

type slogWriter struct{ *slog.Logger }

func (sw slogWriter) Write(p []byte) (int, error) {
	p = bytes.TrimSpace(p)
	sw.Log(context.Background(), slog.LevelInfo, string(p))
	return len(p), nil
}
