package main

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2/humacli"

	"github.com/oaiiae/huma-rest-example/api"
	"github.com/oaiiae/huma-rest-example/logger"
)

// Information set at build time.
var (
	title    string //nolint: gochecknoglobals // set at build time
	version  string //nolint: gochecknoglobals // set at build time
	revision string //nolint: gochecknoglobals // set at build time
	created  string //nolint: gochecknoglobals // set at build time
)

type Options struct {
	Logger logger.Options
	Server api.ServerOptions
	api.RouterOptions
}

func main() {
	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {
		logger := logger.New(&options.Logger)
		router := api.NewRouter(&options.RouterOptions, title, version, revision, created, logger)
		server := api.NewServer(&options.Server, router)

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
