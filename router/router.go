package router

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

var buckets = metrics.ExponentialBuckets(1e-3, 5, 6)

func New(title, version string, readiness http.HandlerFunc, writeMetrics func(io.Writer), opts ...func(huma.API)) http.Handler {
	set := metrics.NewSet()
	set.RegisterMetricsWriter(writeMetrics)

	mux := http.NewServeMux()
	mux.HandleFunc("/liveness", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("/readiness", readiness)
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) { set.WritePrometheus(w) })

	root := humago.New(mux, huma.DefaultConfig(title, version))
	api := huma.NewGroup(root, "/api")
	api.UseMiddleware(
		func(ctx huma.Context, next func(huma.Context)) {
			op, start := ctx.Operation(), time.Now()
			next(ctx)
			labels := fmt.Sprintf(`{method="%s",path="%s",status="%d"}`, op.Method, op.Path, ctx.Status())
			set.GetOrCreatePrometheusHistogramExt(`http_request_duration_seconds`+labels, buckets).UpdateDuration(start)
			set.GetOrCreateCounter(`http_requests_total` + labels).Inc()
			slog.Info(fmt.Sprintln(op.Method, op.Path, "from", ctx.RemoteAddr(), ctx.Status(), time.Since(start)))
		},
	)
	for _, opt := range opts {
		opt(api)
	}

	return mux
}
