package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
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
	title    string //nolint: gochecknoglobals // set at build time
	version  string //nolint: gochecknoglobals // set at build time
	revision string //nolint: gochecknoglobals // set at build time
	created  string //nolint: gochecknoglobals // set at build time
)

// Options for the CLI. Pass `--port` or set the `SERVICE_PORT` env var.
type Options struct {
	Port int `help:"Port to listen on" short:"p" default:"8888"`
}

func main() {
	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		buildinfoMetric := fmt.Sprintf("build_info{title=%q,version=%q,revision=%q,created=%q} 1\n",
			title, version, revision, created)
		metriks := metrics.NewSet()
		router := router.New(title, version,
			func(_ http.ResponseWriter, _ *http.Request) {},
			func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(w, buildinfoMetric)
				metriks.WritePrometheus(w)
				metrics.WriteProcessMetrics(w)
			},
			router.OptUseMiddleware(
				accessLog(logger, slog.LevelInfo),
				meterRequests(metriks),
			),
			router.OptGroup("/api",
				router.OptGroup("/greeting", router.OptAutoRegister(&handler.Greeting{})),
				router.OptGroup("/contacts", router.OptAutoRegister(&handler.Contacts{Store: new(store.ContactsInmem)})),
			),
		)
		server := http.Server{
			Addr:              fmt.Sprintf(":%d", options.Port),
			ReadHeaderTimeout: 15 * time.Second,
			Handler:           router,
		}
		hooks.OnStart(func() {
			logger.Info("starting", "title", title, "version", version, "revision", revision, "created", created)
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

func accessLog(logger *slog.Logger, level slog.Level) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		start := time.Now()
		next(ctx)
		logger.LogAttrs(context.Background(), level,
			ctx.Operation().Method+" "+ctx.Operation().Path+" "+ctx.Version().Proto,
			slog.String("from", ctx.RemoteAddr()),
			slog.String("ref", ctx.Header("Referer")),
			slog.String("ua", ctx.Header("User-Agent")),
			slog.Int("status", ctx.Status()),
			slog.Duration("dur", time.Since(start)),
		)
	}
}

func meterRequests(set *metrics.Set) func(huma.Context, func(huma.Context)) {
	type ref struct {
		total     *metrics.Counter
		histogram *metrics.PrometheusHistogram
	}

	refs := sync.Map{}
	refsMu := sync.Mutex{}
	buckets := metrics.ExponentialBuckets(1e-3, 5, 6) //nolint: mnd // arbitrary

	return func(ctx huma.Context, next func(huma.Context)) {
		op, start := ctx.Operation(), time.Now()
		next(ctx)

		key := op.OperationID + http.StatusText(ctx.Status())
		val, ok := refs.Load(key)
		if !ok {
			refsMu.Lock()
			val, ok = refs.Load(key)
			if !ok {
				labels := fmt.Sprintf(`{method=%q,path=%q,status="%d"}`, op.Method, op.Path, ctx.Status())
				val = ref{
					set.NewCounter("http_requests_total" + labels),
					set.NewPrometheusHistogramExt("http_request_duration_seconds"+labels, buckets),
				}
				refs.Store(key, val)
			}
			refsMu.Unlock()
		}
		valref := val.(ref) //nolint: errcheck // always true
		valref.total.Inc()
		valref.histogram.UpdateDuration(start)
	}
}
