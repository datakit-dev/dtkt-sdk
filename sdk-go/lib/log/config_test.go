package log

import (
	"log/slog"
	"os"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultConfig(t *testing.T) {
	tests := []struct {
		name              string
		appEnv            string
		logLevel          string
		logFormat         string
		expectedLevel     slog.Level
		expectedFormat    Format
		expectedSource    bool
		expectedTelemetry bool
		expectedDefault   bool
	}{
		{
			name:              "dev environment",
			appEnv:            "dev",
			expectedLevel:     slog.LevelDebug,
			expectedFormat:    FormatText,
			expectedSource:    true,
			expectedTelemetry: false,
			expectedDefault:   true,
		},
		{
			name:              "test environment",
			appEnv:            "test",
			expectedLevel:     slog.LevelDebug,
			expectedFormat:    FormatText,
			expectedSource:    true,
			expectedTelemetry: false,
			expectedDefault:   true,
		},
		{
			name:              "seed environment",
			appEnv:            "seed",
			expectedLevel:     slog.LevelDebug,
			expectedFormat:    FormatText,
			expectedSource:    true,
			expectedTelemetry: false,
			expectedDefault:   true,
		},
		{
			name:              "prd environment",
			appEnv:            "prd",
			expectedLevel:     slog.LevelInfo,
			expectedFormat:    FormatJSON,
			expectedSource:    false,
			expectedTelemetry: false,
			expectedDefault:   true,
		},
		{
			name:              "default environment",
			appEnv:            "unknown",
			expectedLevel:     slog.LevelInfo,
			expectedFormat:    FormatJSON,
			expectedSource:    false,
			expectedTelemetry: false,
			expectedDefault:   true,
		},
		{
			name:              "override with env vars",
			appEnv:            "prd",
			logLevel:          "DEBUG",
			logFormat:         "TEXT",
			expectedLevel:     slog.LevelDebug,
			expectedFormat:    FormatText,
			expectedSource:    false,
			expectedTelemetry: false,
			expectedDefault:   true,
		},
		{
			name:              "override with different log level",
			appEnv:            "dev",
			logLevel:          "ERROR",
			expectedLevel:     slog.LevelError,
			expectedFormat:    FormatText,
			expectedSource:    true,
			expectedTelemetry: false,
			expectedDefault:   true,
		},
		{
			name:              "override with json format",
			appEnv:            "dev",
			logFormat:         "JSON",
			expectedLevel:     slog.LevelDebug,
			expectedFormat:    FormatJSON,
			expectedSource:    true,
			expectedTelemetry: false,
			expectedDefault:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment

			//nolint:errcheck
			os.Unsetenv(env.AppEnv)
			//nolint:errcheck
			os.Unsetenv(env.LogLevel)
			//nolint:errcheck
			os.Unsetenv(env.LogFormat)

			// Set test environment
			if tt.appEnv != "" {
				//nolint:errcheck
				os.Setenv(env.AppEnv, tt.appEnv)
			}
			if tt.logLevel != "" {
				//nolint:errcheck
				os.Setenv(env.LogLevel, tt.logLevel)
			}
			if tt.logFormat != "" {
				//nolint:errcheck
				os.Setenv(env.LogFormat, tt.logFormat)
			}

			// Get config
			config := GetDefaultConfig()

			// Assert
			assert.Equal(t, tt.expectedLevel, config.Level, "Level should match")
			assert.Equal(t, tt.expectedFormat, config.Format, "Format should match")
			assert.Equal(t, tt.expectedSource, config.IncludeSource, "IncludeSource should match")
			assert.Equal(t, tt.expectedTelemetry, config.Telemetry, "Telemetry should match")
			assert.Equal(t, tt.expectedDefault, config.SlogDefault, "SlogDefault should match")
			assert.Empty(t, config.FilePath, "FilePath should be empty by default")

			// Clean up
			//nolint:errcheck
			os.Unsetenv(env.AppEnv)
			//nolint:errcheck
			os.Unsetenv(env.LogLevel)
			//nolint:errcheck
			os.Unsetenv(env.LogFormat)
		})
	}
}

func TestWithLevel(t *testing.T) {
	config := &Config{}
	option := WithLevel(slog.LevelWarn)
	option(config)

	assert.Equal(t, slog.LevelWarn, config.Level)
}

func TestWithLogLevel(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedLevel slog.Level
	}{
		{
			name:          "debug level",
			level:         "DEBUG",
			expectedLevel: slog.LevelDebug,
		},
		{
			name:          "info level",
			level:         "INFO",
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "warn level",
			level:         "WARN",
			expectedLevel: slog.LevelWarn,
		},
		{
			name:          "error level",
			level:         "ERROR",
			expectedLevel: slog.LevelError,
		},
		{
			name:          "lowercase level",
			level:         "debug",
			expectedLevel: slog.LevelDebug,
		},
		{
			name:          "mixed case level",
			level:         "WaRn",
			expectedLevel: slog.LevelWarn,
		},
		{
			name:          "unknown level defaults to info",
			level:         "UNKNOWN",
			expectedLevel: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithLogLevel(tt.level)
			option(config)

			assert.Equal(t, tt.expectedLevel, config.Level)
		})
	}
}

func TestWithLogFormat(t *testing.T) {
	tests := []struct {
		name           string
		format         string
		expectedFormat Format
	}{
		{
			name:           "text format",
			format:         "TEXT",
			expectedFormat: FormatText,
		},
		{
			name:           "json format",
			format:         "JSON",
			expectedFormat: FormatJSON,
		},
		{
			name:           "lowercase format",
			format:         "text",
			expectedFormat: FormatText,
		},
		{
			name:           "mixed case format",
			format:         "JsOn",
			expectedFormat: FormatJSON,
		},
		{
			name:           "unknown format defaults to json",
			format:         "UNKNOWN",
			expectedFormat: FormatJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithLogFormat(tt.format)
			option(config)

			assert.Equal(t, tt.expectedFormat, config.Format)
		})
	}
}

func TestWithFormat(t *testing.T) {
	tests := []struct {
		name   string
		format Format
	}{
		{
			name:   "text format",
			format: FormatText,
		},
		{
			name:   "json format",
			format: FormatJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithFormat(tt.format)
			option(config)

			assert.Equal(t, tt.format, config.Format)
		})
	}
}

func TestWithSource(t *testing.T) {
	tests := []struct {
		name    string
		include bool
	}{
		{
			name:    "include source",
			include: true,
		},
		{
			name:    "exclude source",
			include: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithSource(tt.include)
			option(config)

			assert.Equal(t, tt.include, config.IncludeSource)
		})
	}
}

func TestWithTelemetry(t *testing.T) {
	tests := []struct {
		name      string
		telemetry bool
	}{
		{
			name:      "enable telemetry",
			telemetry: true,
		},
		{
			name:      "disable telemetry",
			telemetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithTelemetry(tt.telemetry)
			option(config)

			assert.Equal(t, tt.telemetry, config.Telemetry)
		})
	}
}

func TestWithSlogDefault(t *testing.T) {
	tests := []struct {
		name        string
		slogDefault bool
	}{
		{
			name:        "set as default",
			slogDefault: true,
		},
		{
			name:        "not set as default",
			slogDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithSlogDefault(tt.slogDefault)
			option(config)

			assert.Equal(t, tt.slogDefault, config.SlogDefault)
		})
	}
}

func TestWithFile(t *testing.T) {
	filePath := "/tmp/test.log"
	config := &Config{}
	option := WithFile(filePath)
	option(config)

	assert.Equal(t, filePath, config.FilePath)
}

func TestConfig_IsFileLogger(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "with file path",
			filePath: "/tmp/test.log",
			expected: true,
		},
		{
			name:     "without file path",
			filePath: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				FilePath: tt.filePath,
			}

			assert.Equal(t, tt.expected, config.IsFileLogger())
		})
	}
}

func TestToFormat(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedFormat Format
	}{
		{
			name:           "uppercase TEXT",
			input:          "TEXT",
			expectedFormat: FormatText,
		},
		{
			name:           "lowercase text",
			input:          "text",
			expectedFormat: FormatText,
		},
		{
			name:           "mixed case Text",
			input:          "Text",
			expectedFormat: FormatText,
		},
		{
			name:           "uppercase JSON",
			input:          "JSON",
			expectedFormat: FormatJSON,
		},
		{
			name:           "lowercase json",
			input:          "json",
			expectedFormat: FormatJSON,
		},
		{
			name:           "mixed case Json",
			input:          "Json",
			expectedFormat: FormatJSON,
		},
		{
			name:           "unknown format defaults to JSON",
			input:          "UNKNOWN",
			expectedFormat: FormatJSON,
		},
		{
			name:           "empty string defaults to JSON",
			input:          "",
			expectedFormat: FormatJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toFormat(tt.input)
			assert.Equal(t, tt.expectedFormat, result)
		})
	}
}

func TestToLogLevel(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedLevel slog.Level
	}{
		{
			name:          "uppercase DEBUG",
			input:         "DEBUG",
			expectedLevel: slog.LevelDebug,
		},
		{
			name:          "lowercase debug",
			input:         "debug",
			expectedLevel: slog.LevelDebug,
		},
		{
			name:          "mixed case Debug",
			input:         "Debug",
			expectedLevel: slog.LevelDebug,
		},
		{
			name:          "uppercase INFO",
			input:         "INFO",
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "lowercase info",
			input:         "info",
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "uppercase WARN",
			input:         "WARN",
			expectedLevel: slog.LevelWarn,
		},
		{
			name:          "lowercase warn",
			input:         "warn",
			expectedLevel: slog.LevelWarn,
		},
		{
			name:          "uppercase ERROR",
			input:         "ERROR",
			expectedLevel: slog.LevelError,
		},
		{
			name:          "lowercase error",
			input:         "error",
			expectedLevel: slog.LevelError,
		},
		{
			name:          "unknown level defaults to INFO",
			input:         "UNKNOWN",
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "empty string defaults to INFO",
			input:         "",
			expectedLevel: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toLogLevel(tt.input)
			assert.Equal(t, tt.expectedLevel, result)
		})
	}
}

func TestMultipleOptions(t *testing.T) {
	config := &Config{}

	options := []Option{
		WithLevel(slog.LevelWarn),
		WithFormat(FormatText),
		WithSource(true),
		WithTelemetry(true),
		WithSlogDefault(false),
		WithFile("/tmp/test.log"),
	}

	for _, opt := range options {
		opt(config)
	}

	assert.Equal(t, slog.LevelWarn, config.Level)
	assert.Equal(t, FormatText, config.Format)
	assert.True(t, config.IncludeSource)
	assert.True(t, config.Telemetry)
	assert.False(t, config.SlogDefault)
	assert.Equal(t, "/tmp/test.log", config.FilePath)
}

func TestNilOption(t *testing.T) {
	config := &Config{
		Level: slog.LevelInfo,
	}

	var nilOption Option = nil
	require.NotPanics(t, func() {
		if nilOption != nil {
			nilOption(config)
		}
	})

	// Config should remain unchanged
	assert.Equal(t, slog.LevelInfo, config.Level)
}
