package logger

import (
	"io"
	"log/slog"
	"os"
)

// Options for the CLI.
type Options struct {
	Output string `doc:"path to logfile" default:"-"`
	Level  string `doc:"minimum log level" default:"info"`
	Format string `doc:"log format" default:"text"`
}

func New(options *Options) *slog.Logger {
	var err error

	var output io.Writer
	switch options.Output {
	case "-":
		output = os.Stdout
	case os.DevNull:
		return slog.New(slog.DiscardHandler)
	default:
		output, err = os.Open(options.Output)
		if err != nil {
			options.Output = "-"
			logger := New(options)
			logger.Warn("could not open file", "err", err)
			return logger
		}
	}

	var level slog.Level
	err = level.UnmarshalText([]byte(options.Level))
	if err != nil {
		options.Level = "info"
		logger := New(options)
		logger.Warn("could not parse level", "err", err)
		return logger
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	switch options.Format {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	case "text":
		handler = slog.NewTextHandler(output, opts)
	default:
		options.Format = "text"
		logger := New(options)
		logger.Warn("unknown format")
		return logger
	}

	return slog.New(handler)
}
