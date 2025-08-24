package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

type Options struct {
	Level   string `doc:"minimum log level" default:""`
	Logfile string `doc:"write logs to file" default:""`
	Format  string `doc:"log format" default:"text"`
}

func New(options *Options) *slog.Logger {
	var err error

	var opts slog.HandlerOptions
	switch strings.ToLower(options.Level) {
	case "":
		opts.Level = nil
	case "debug":
		opts.Level = slog.LevelDebug
	case "info":
		opts.Level = slog.LevelInfo
	case "warn":
		opts.Level = slog.LevelWarn
	case "error":
		opts.Level = slog.LevelError
	default:
		options.Level = ""
		logger := New(options)
		logger.Warn("could not parse logger level", "err", err)
		return logger
	}

	var output io.Writer
	switch options.Logfile {
	case "":
		output = os.Stdout
	case os.DevNull:
		return slog.New(slog.DiscardHandler)
	default:
		output, err = os.Open(options.Logfile)
		if err != nil {
			options.Logfile = ""
			logger := New(options)
			logger.Warn("could not open logger output", "err", err)
			return logger
		}
	}

	var handler slog.Handler
	switch strings.ToLower(options.Format) {
	case "json":
		handler = slog.NewJSONHandler(output, &opts)
	case "text":
		handler = slog.NewTextHandler(output, &opts)
	default:
		options.Format = "text"
		logger := New(options)
		logger.Warn("could not parse logger format")
		return logger
	}

	return slog.New(handler)
}
