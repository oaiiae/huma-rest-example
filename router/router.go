package router

import (
	"fmt"
	"io"
	"net/http"
	"sync"
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

	metricsMap := sync.Map{}
	metricsMu := sync.Mutex{}
	metricsMiddleware := func(ctx huma.Context, next func(huma.Context)) {
		op, start := ctx.Operation(), time.Now()
		next(ctx)

		type metricsMapValue struct {
			http_requests_total           *metrics.Counter
			http_request_duration_seconds *metrics.PrometheusHistogram
		}

		key := op.OperationID + http.StatusText(ctx.Status())
		val, ok := metricsMap.Load(key)
		if !ok {
			metricsMu.Lock()
			val, ok = metricsMap.Load(key)
			if !ok {
				labels := fmt.Sprintf(`{method=%q,path=%q,status="%d"}`, op.Method, op.Path, ctx.Status())
				val = metricsMapValue{
					set.NewCounter("http_requests_total" + labels),
					set.NewPrometheusHistogramExt("http_request_duration_seconds"+labels, metrics.ExponentialBuckets(1e-3, 5, 6)),
				}
				metricsMap.Store(key, val)
			}
			metricsMu.Unlock()
		}
		val.(metricsMapValue).http_requests_total.Inc()
		val.(metricsMapValue).http_request_duration_seconds.UpdateDuration(start)
	}

	api := humago.New(mux, huma.DefaultConfig(title, version))
	api.UseMiddleware(metricsMiddleware)
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
