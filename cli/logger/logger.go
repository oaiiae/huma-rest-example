package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

type Options struct {
	Level  string `doc:"log from debug, info, warn or error"`
	File   string `doc:"append logs to file"`
	Format string `doc:"format logs as text or json"         default:"text"`
}

func level(option string) (slog.Leveler, bool) {
	switch strings.ToLower(option) {
	case "":
		return nil, true
	case "debug":
		return slog.LevelDebug, true
	case "info":
		return slog.LevelInfo, true
	case "warn":
		return slog.LevelWarn, true
	case "error":
		return slog.LevelError, true
	default:
		return nil, false
	}
}

func New(options *Options) *slog.Logger {
	level, ok := level(options.Level)
	if !ok {
		options.Level = ""
		logger := New(options)
		logger.Warn("could not parse logger level")
		return logger
	}
	opts := slog.HandlerOptions{Level: level}

	var output io.Writer
	switch options.File {
	case "", "-":
		output = os.Stdout
	case os.DevNull:
		return slog.New(slog.DiscardHandler)
	default:
		var err error
		output, err = os.OpenFile(options.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			options.File = ""
			logger := New(options)
			logger.Warn("could not open logger file", "err", err)
			return logger
		}
	}

	switch strings.ToLower(options.Format) {
	case "json":
		return slog.New(slog.NewJSONHandler(output, &opts))
	case "text":
		return slog.New(slog.NewTextHandler(output, &opts))
	default:
		options.Format = "text"
		logger := New(options)
		logger.Warn("could not parse logger format")
		return logger
	}
}
