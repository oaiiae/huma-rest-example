package main

import (
	"context"
	"fmt"
	"io"
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

// Information set at build time.
var (
	title    string
	version  string
	revision string
	created  string
)

func buildinfoMetricWriter() func(io.Writer) {
	metric := fmt.Sprintf(`build_info{title="%s",version="%s",revision="%s",created="%s"} 1`+"\n", title, version, revision, created)
	return func(w io.Writer) { w.Write([]byte(metric)) }
}

// Options for the CLI. Pass `--port` or set the `SERVICE_PORT` env var.
type Options struct {
	Port int `help:"Port to listen on" short:"p" default:"8888"`
}

func main() {
	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		server := http.Server{
			Addr: fmt.Sprintf(":%d", options.Port),
			Handler: router.New(title, version,
				func(w http.ResponseWriter, r *http.Request) {},
				router.MetricsWriters(metrics.WriteProcessMetrics, buildinfoMetricWriter()),
				router.OptUseMiddleware(accesslog(logger, slog.LevelInfo)),
				router.OptGroup("/api",
					router.OptGroup("/greeting", router.OptAutoRegister(&handler.Greeting{})),
					router.OptGroup("/contacts", router.OptAutoRegister(&handler.Contacts{Store: new(store.ContactsInmem)})),
				),
			),
			ReadHeaderTimeout: 15 * time.Second,
		}
		hooks.OnStart(func() {
			logger.Info("server starts", "title", title, "version", version, "revision", revision, "created", created)
			err := server.ListenAndServe()
			if err != http.ErrServerClosed {
				logger.Error("server failure", "err", err)
			} else {
				logger.Info("server closed")
			}
		})
		hooks.OnStop(func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			err := server.Shutdown(ctx)
			if err != nil {
				logger.Warn("could not shutdown the server", "err", err)
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
