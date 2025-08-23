package router

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func New(
	title, version string,
	readiness http.HandlerFunc,
	metrics http.HandlerFunc,
	opts ...func(huma.API),
) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/liveness", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("/readiness", readiness)
	mux.HandleFunc("/metrics", metrics)

	api := humago.New(mux, huma.DefaultConfig(title, version))
	for _, opt := range opts {
		opt(api)
	}

	return mux
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
