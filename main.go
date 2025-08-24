package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
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
		buildinfoMetric := joinQuote("build_info{title=", title, ",version=", version, ",revision=", revision, ",created=", created, "} 1\n")
		metriks := metrics.NewSet()
		router := router.New(title, version,
			func(_ http.ResponseWriter, _ *http.Request) {},
			func(w http.ResponseWriter, _ *http.Request) {
				w.Write([]byte(buildinfoMetric))
				metriks.WritePrometheus(w)
				metrics.WriteProcessMetrics(w)
			},
			router.OptUseMiddleware(
				ctxlog{}.loggerMiddleware(logger, slog.LevelInfo),
				meterRequests(metriks),
				ctxlog{}.recoverMiddleware(logger, slog.LevelError),
			),
			router.OptGroup("/api",
				router.OptGroup("/panic", router.OptAutoRegister(&handler.Panic{})),
				router.OptGroup("/greeting", router.OptAutoRegister(&handler.Greeting{})),
				router.OptGroup("/contacts", router.OptAutoRegister(&handler.Contacts{
					Store: new(store.ContactsInmem).With(store.Contact{
						ID:        12,
						Firstname: "john",
						Lastname:  "smith",
						Birthday:  time.Date(1999, time.December, 31, 0, 0, 0, 0, time.UTC),
					}),
					ErrorHandler: ctxlog{}.errorHandler(logger, "/contacts"),
				})),
			),
		)
		server := http.Server{
			Addr:              ":" + strconv.Itoa(options.Port),
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

// ctxlog is a [context.Context] key and acts as a virtual package for operations related to it.
type ctxlog struct{}

// loggerMiddleware returns a middleware that sets a [slog.Logger] in
// the [context.Context] and logs the request after it has terminated.
func (key ctxlog) loggerMiddleware(parent *slog.Logger, level slog.Level) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		logger := parent.With("x-request-id", ctx.Header("X-Request-Id"))

		start := time.Now()
		next(huma.WithValue(ctx, key, logger.WithGroup("op").With("id", ctx.Operation().OperationID)))

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

// recoverMiddleware returns a middleware that recovers and logs the value from panic.
func (key ctxlog) recoverMiddleware(fallback *slog.Logger, level slog.Level) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		defer func() {
			v := recover()
			if v != nil {
				logger, ok := ctx.Context().Value(key).(*slog.Logger)
				if !ok {
					logger = fallback
				}
				logger.LogAttrs(context.Background(), level, "panic occured", slog.Any("recovered", v))
				ctx.SetStatus(http.StatusInternalServerError)
			}
		}()
		next(ctx)
	}
}

// errorHandler returns a function that gets the [slog.Logger] in the [context.Context]
// using [ctxlog] as key and logs the error.
func (key ctxlog) errorHandler(fallback *slog.Logger, msg string) func(context.Context, error) {
	return func(ctx context.Context, err error) {
		logger, ok := ctx.Value(key).(*slog.Logger)
		if !ok {
			logger = fallback
		}

		level := slog.LevelError
		attrs := []slog.Attr{slog.String("err", err.Error())}

		var statusErr huma.StatusError
		if errors.As(err, &statusErr) {
			switch statusErr.GetStatus() / 100 {
			case 5:
				level = slog.LevelError
			case 4:
				level = slog.LevelWarn
			case 3:
				level = slog.LevelInfo
			}
			attrs = append(attrs, slog.Int("status", statusErr.GetStatus()))
		}

		logger.LogAttrs(context.Background(), level, msg, attrs...)
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

		uid := op.OperationID + http.StatusText(ctx.Status())
		val, ok := refs.Load(uid)
		if !ok {
			refsMu.Lock()
			val, ok = refs.Load(uid)
			if !ok {
				labels := joinQuote("{method=", op.Method, ",path=", op.Path, ",status=", strconv.Itoa(ctx.Status()), "}")
				val = ref{
					set.NewCounter("http_requests_total" + labels),
					set.NewPrometheusHistogramExt("http_request_duration_seconds"+labels, buckets),
				}
				refs.Store(uid, val)
			}
			refsMu.Unlock()
		}
		valref := val.(ref) //nolint: errcheck // always true
		valref.total.Inc()
		valref.histogram.UpdateDuration(start)
	}
}

// joinQuote is [strings.Join] with " as separator.
func joinQuote(elems ...string) string { return strings.Join(elems, `"`) }
