package log

import (
	"log/slog"
	"os"
	"path/filepath"

	charmLog "github.com/charmbracelet/log"
)

type Format string

var logFiles map[*slog.Logger]*os.File = make(map[*slog.Logger]*os.File)

const (
	FormatText   Format = "text"
	FormatJSON   Format = "json"
	FormatPretty Format = "pretty"
)

func CloseLogger(logger *slog.Logger) error {
	if logFiles[logger] != nil {
		err := logFiles[logger].Close()

		if err != nil {
			return err
		}

		delete(logFiles, logger)
	}

	return nil
}

func NewLogger(options ...Option) *slog.Logger {
	config := GetDefaultConfig()

	for _, option := range options {
		if option != nil {
			option(config)
		}
	}

	w := os.Stderr

	if config.IsFileLogger() {
		err := os.MkdirAll(filepath.Dir(config.FilePath), 0o755)
		if err != nil {
			panic(err)
		}

		w, err = os.OpenFile(config.FilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
		if err != nil {
			panic(err)
		}
	}

	handlerOptions := slog.HandlerOptions{
		AddSource: config.IncludeSource,
		Level:     config.Level,
	}

	var handler slog.Handler
	switch config.Format {
	case FormatText:
		handler = slog.NewTextHandler(w, &handlerOptions)
	case FormatJSON:
		handler = slog.NewJSONHandler(w, &handlerOptions)
	case FormatPretty:
		handler = charmLog.NewWithOptions(w, charmLog.Options{
			Level: toCharmLogLevel(handlerOptions.Level.Level()),
		})
	}

	logger := slog.New(handler)

	if config.IsFileLogger() {
		logFiles[logger] = w
	}

	if config.SlogDefault {
		slog.SetDefault(logger)
	}

	return logger
}

func toCharmLogLevel(level slog.Level) charmLog.Level {
	switch level {
	case slog.LevelDebug:
		return charmLog.DebugLevel
	case slog.LevelInfo:
		return charmLog.InfoLevel
	case slog.LevelWarn:
		return charmLog.WarnLevel
	case slog.LevelError:
		return charmLog.ErrorLevel
	case 12:
		return charmLog.FatalLevel
	default:
		return charmLog.InfoLevel
	}
}
