package router

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

var buckets = metrics.ExponentialBuckets(1e-3, 5, 6)

func New(
	title, version string,
	readiness http.HandlerFunc,
	writeMetrics func(io.Writer),
	opts ...func(huma.API),
) http.Handler {
	set := metrics.NewSet()
	set.RegisterMetricsWriter(writeMetrics)

	mux := http.NewServeMux()
	mux.HandleFunc("/liveness", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("/readiness", readiness)
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) { set.WritePrometheus(w) })

	api := humago.New(mux, huma.DefaultConfig(title, version))
	api.UseMiddleware(
		func(ctx huma.Context, next func(huma.Context)) {
			op, start := ctx.Operation(), time.Now()
			next(ctx)
			labels := fmt.Sprintf(`{method="%s",path="%s",status="%d"}`, op.Method, op.Path, ctx.Status())
			set.GetOrCreatePrometheusHistogramExt(`http_request_duration_seconds`+labels, buckets).UpdateDuration(start)
			set.GetOrCreateCounter(`http_requests_total` + labels).Inc()
		},
	)
	for _, opt := range opts {
		opt(api)
	}

	return mux
}

func MetricsWriters(writers ...func(io.Writer)) func(io.Writer) {
	return func(w io.Writer) {
		for _, writer := range writers {
			writer(w)
		}
	}
}

func OptUseMiddleware(middlewares ...func(huma.Context, func(huma.Context))) func(huma.API) {
	return func(api huma.API) { api.UseMiddleware(middlewares...) }
}

func OptAutoRegister(server any) func(huma.API) {
	return func(api huma.API) { huma.AutoRegister(api, server) }
}

type Prefixes []string

func (p Prefixes) OptGroup(opts ...func(huma.API)) func(huma.API) {
	return func(api huma.API) {
		g := huma.NewGroup(api, p...)
		for _, opt := range opts {
			opt(g)
		}
	}
}

func OptGroup(prefix string, opts ...func(huma.API)) func(huma.API) {
	return Prefixes{prefix}.OptGroup(opts...)
}
