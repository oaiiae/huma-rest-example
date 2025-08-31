package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/danielgtaylor/huma/v2"

	"github.com/oaiiae/huma-rest-example/datastores"
	"github.com/oaiiae/huma-rest-example/handlers"
	"github.com/oaiiae/huma-rest-example/router"
)

type ServerOptions struct {
	Host              string        `short:"H" doc:"host to listen on"                    default:""`
	Port              string        `short:"p" doc:"port to listen on"                    default:"8888"`
	ReadHeaderTimeout time.Duration `          doc:"time allowed to read request headers" default:"15s"`
}

func NewServer(options *ServerOptions, handler http.Handler, logger *slog.Logger) *http.Server {
	return &http.Server{
		Addr:              options.Host + ":" + options.Port,
		ReadHeaderTimeout: options.ReadHeaderTimeout,
		Handler:           handler,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
}

type RouterOptions struct {
	EndpointsPrefix string `doc:"mount endpoints at a prefix" default:"/api"`
}

func NewRouter(
	options *RouterOptions,
	title string,
	version string,
	revision string,
	created string,
	logger *slog.Logger,
) http.Handler {
	buildinfoMetric := joinQuote("build_info{goversion=", runtime.Version(),
		",title=", title,
		",version=", version,
		",revision=", revision,
		",created=", created,
		"} 1\n")
	metriks := metrics.NewSet()
	return router.New(title, version,
		func(_ http.ResponseWriter, _ *http.Request) {},
		func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, buildinfoMetric)
			metriks.WritePrometheus(w)
			metrics.WriteProcessMetrics(w)
		},
		router.OptUseMiddleware(
			ctxlog{}.loggerMiddleware(logger),
			meterRequestsMiddleware(metriks),
			meterRequestsStatusMiddleware(metriks),
			ctxlog{}.recoverMiddleware(logger),
		),
		router.OptGroup(options.EndpointsPrefix,
			router.OptGroup("/panic", router.OptAutoRegister(&handlers.Panic{})),
			router.OptGroup("/greeting", router.OptAutoRegister(&handlers.Greeting{})),
			router.OptGroup("/contacts", router.OptAutoRegister(&handlers.Contacts{
				Store: datastores.NewContactsInmem(&datastores.Contact{
					Firstname: "john",
					Lastname:  "smith",
					Birthday:  time.Date(1999, time.December, 31, 0, 0, 0, 0, time.UTC),
				}),
				ErrorHandler: ctxlog{}.errorHandler(logger),
			})),
		),
	)
}

// ctxlog is a [context.Context] key and acts as a virtual package for operations related to it.
type ctxlog struct{}

// loggerMiddleware returns a middleware that sets a [slog.Logger] in
// the [context.Context] and logs the request after it has terminated.
func (key ctxlog) loggerMiddleware(parent *slog.Logger) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		logger := parent.With("x-request-id", ctx.Header("X-Request-Id"))

		start := time.Now()
		next(huma.WithValue(ctx, key, logger.WithGroup("op").With("id", ctx.Operation().OperationID)))

		logger.LogAttrs(context.Background(), slog.LevelInfo,
			joinSpace(ctx.Operation().Method, ctx.Operation().Path, ctx.Version().Proto),
			slog.String("from", ctx.RemoteAddr()),
			slog.String("ref", ctx.Header("Referer")),
			slog.String("ua", ctx.Header("User-Agent")),
			slog.Int("status", ctx.Status()),
			slog.Duration("dur", time.Since(start)),
		)
	}
}

// recoverMiddleware returns a middleware that recovers and logs the value from panic
// to finally set the response status to [http.StatusInternalServerError].
func (key ctxlog) recoverMiddleware(fallback *slog.Logger) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		defer func() {
			v := recover()
			if v != nil {
				logger, ok := ctx.Context().Value(key).(*slog.Logger)
				if !ok {
					logger = fallback
				}
				logger.LogAttrs(context.Background(), slog.LevelError, "panic occurred", slog.Any("recovered", v))
				ctx.SetStatus(http.StatusInternalServerError)
			}
		}()
		next(ctx)
	}
}

// errorHandler returns a function that gets the [slog.Logger] from [context.Context] and logs the error.
func (key ctxlog) errorHandler(fallback *slog.Logger) func(context.Context, error) {
	return func(ctx context.Context, err error) {
		level := slog.LevelError
		attrs := []slog.Attr{slog.Any("err", err)}

		var statusErr huma.StatusError
		if errors.As(err, &statusErr) {
			switch statusErr.GetStatus() / 100 {
			case 5: //nolint: mnd // 5XX HTTP Status Codes
				level = slog.LevelError
			case 4: //nolint: mnd // 4XX HTTP Status Codes
				level = slog.LevelWarn
			case 3: //nolint: mnd // 3XX HTTP Status Codes
				level = slog.LevelInfo
			}
			attrs = append(attrs, slog.Int("status", statusErr.GetStatus()))
		}

		logger, ok := ctx.Value(key).(*slog.Logger)
		if !ok {
			logger = fallback
		}
		logger.LogAttrs(context.Background(), level, "error occurred", attrs...)
	}
}

// meterRequestsMiddleware returns a middleware registering metrics about requests.
//
//   - http_requests_in_flight{method,path}
func meterRequestsMiddleware(set *metrics.Set) func(huma.Context, func(huma.Context)) {
	smap := sync.Map{}
	return func(ctx huma.Context, next func(huma.Context)) {
		op := ctx.Operation()
		k := op.OperationID
		v, ok := smap.Load(k)
		if !ok {
			labels := joinQuote("{method=", op.Method, ",path=", op.Path, "}")
			v, _ = smap.LoadOrStore(k,
				set.GetOrCreateCounter("http_requests_in_flight"+labels),
			)
		}
		val := v.(*metrics.Counter) //nolint: errcheck // always true
		val.Inc()
		defer val.Dec()

		next(ctx)
	}
}

// meterRequestsStatusMiddleware returns a middleware registering metrics about requests and their response status.
//
//   - http_request_duration_seconds_bucket{method,path,status,le}
//   - http_request_duration_seconds_sum{method,path,status}
//   - http_request_duration_seconds_count{method,path,status}
//   - http_requests_total{method,path,status}
func meterRequestsStatusMiddleware(set *metrics.Set) func(huma.Context, func(huma.Context)) {
	type value struct {
		*metrics.PrometheusHistogram
		*metrics.Counter
	}
	var buckets = metrics.ExponentialBuckets(1e-3, 5, 6) //nolint: mnd // arbitrary

	smap := sync.Map{}
	return func(ctx huma.Context, next func(huma.Context)) {
		start := time.Now()
		next(ctx)

		op := ctx.Operation()
		k := op.OperationID + http.StatusText(ctx.Status())
		v, ok := smap.Load(k)
		if !ok {
			labels := joinQuote("{method=", op.Method, ",path=", op.Path, ",status=", strconv.Itoa(ctx.Status()), "}") //nolint: golines
			v, _ = smap.LoadOrStore(k, value{
				set.GetOrCreatePrometheusHistogramExt("http_request_duration_seconds"+labels, buckets),
				set.GetOrCreateCounter("http_requests_total" + labels),
			})
		}
		val := v.(value) //nolint: errcheck // always true
		val.PrometheusHistogram.UpdateDuration(start)
		val.Counter.Inc()
	}
}

// joinQuote is [strings.Join] with " as separator.
func joinQuote(elems ...string) string { return strings.Join(elems, `"`) }

// joinSpace is [strings.Join] with space as separator.
func joinSpace(elems ...string) string { return strings.Join(elems, ` `) }
