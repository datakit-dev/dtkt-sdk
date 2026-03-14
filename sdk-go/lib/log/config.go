package log

import (
	"log/slog"
	"os"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/env"
)

type Option func(*Config)

type Config struct {
	Level  slog.Level
	Format Format

	IncludeSource bool

	// If true, telemetry logs will be enabled
	// These logs will stream to an OTEL logs collector
	// TODO: Implement telemetry logging
	Telemetry bool

	// If true the logger will be set as the default logger
	SlogDefault bool

	// FilePath is the path to the log file if logging to a file
	FilePath string
}

func GetDefaultConfig() *Config {
	var level slog.Level
	var format Format

	envLevel, isEnvLevelSet := os.LookupEnv(env.LogLevel)
	envFormat, isEnvFormatSet := os.LookupEnv(env.LogFormat)
	envSource, isEnvSourceSet := os.LookupEnv(env.LogSource)

	appEnv := os.Getenv(env.AppEnv)

	var includeSource bool

	switch appEnv {
	case "test", "seed", "dev":
		level = slog.LevelDebug
		format = FormatText
		includeSource = true
	case "prd":
		fallthrough
	default:
		level = slog.LevelInfo
		format = FormatJSON
		includeSource = false
	}

	if isEnvLevelSet {
		level = toLogLevel(envLevel)
	}

	if isEnvFormatSet {
		format = toFormat(envFormat)
	}

	if isEnvSourceSet {
		includeSource = strings.ToLower(envSource) == "true"
	}

	return &Config{
		Level:  level,
		Format: format,

		IncludeSource: includeSource,

		Telemetry: false,

		SlogDefault: true,
	}
}

func WithLevel(level slog.Level) Option {
	return func(c *Config) {
		c.Level = level
	}
}

func WithLogLevel(level string) Option {
	return func(c *Config) {
		c.Level = toLogLevel(level)
	}
}

func WithLogFormat(format string) Option {
	return func(c *Config) {
		c.Format = toFormat(format)
	}
}

func WithFormat(format Format) Option {
	return func(c *Config) {
		c.Format = format
	}
}

func WithSource(include bool) Option {
	return func(c *Config) {
		c.IncludeSource = include
	}
}

func WithTelemetry(telemetry bool) Option {
	return func(c *Config) {
		c.Telemetry = telemetry
	}
}

func WithSlogDefault(slogDefault bool) Option {
	return func(c *Config) {
		c.SlogDefault = slogDefault
	}
}

func WithFile(filePath string) Option {
	return func(c *Config) {
		c.FilePath = filePath
	}
}

func (c *Config) IsFileLogger() bool {
	return c.FilePath != ""
}

func toFormat(format string) Format {
	switch strings.ToLower(format) {
	case "text":
		return FormatText
	case "json":
		return FormatJSON
	case "pretty":
		return FormatPretty
	default:
		return FormatJSON
	}
}

// func toFormatText(format Format) string {
// 	switch format {
// 	case FormatText:
// 		return "text"
// 	case FormatJSON:
// 		return "json"
// 	case FormatPretty:
// 		return "pretty"
// 	default:
// 		return "json"
// 	}
// }

func toLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// func toLogLevelText(level slog.Level) string {
// 	switch level {
// 	case slog.LevelDebug:
// 		return "DEBUG"
// 	case slog.LevelInfo:
// 		return "INFO"
// 	case slog.LevelWarn:
// 		return "WARN"
// 	case slog.LevelError:
// 		return "ERROR"
// 	default:
// 		return "INFO"
// 	}
// }
