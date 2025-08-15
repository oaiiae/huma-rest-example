package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/danielgtaylor/huma/v2"
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
				router.OptUseMiddleware(accesslog(logger, slog.LevelInfo)),
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

func accesslog(l *slog.Logger, level slog.Level) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		next(ctx)
		l.LogAttrs(context.Background(), level,
			ctx.Operation().Method+" "+ctx.Operation().Path+" "+ctx.Version().Proto,
			slog.String("from", ctx.RemoteAddr()),
			slog.Int("status", ctx.Status()),
			slog.String("ref", ctx.Header("Referer")),
			slog.String("ua", ctx.Header("User-Agent")),
		)
	}
}
