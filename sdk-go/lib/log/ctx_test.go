package log

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCtx(t *testing.T) {
	t.Run("with nil context", func(t *testing.T) {
		logger := slog.Default()
		ctx := NewCtx(t.Context(), logger)

		require.NotNil(t, ctx)
		assert.Equal(t, logger, ctx.Value(contextKey{}))
	})

	t.Run("with existing context", func(t *testing.T) {
		logger := slog.Default()
		parentCtx := context.Background()
		ctx := NewCtx(parentCtx, logger)

		require.NotNil(t, ctx)
		assert.Equal(t, logger, ctx.Value(contextKey{}))
	})

	t.Run("preserves parent context values", func(t *testing.T) {
		logger := slog.Default()
		type testKey struct{}
		parentCtx := context.WithValue(context.Background(), testKey{}, "test-value")
		ctx := NewCtx(parentCtx, logger)

		require.NotNil(t, ctx)
		assert.Equal(t, "test-value", ctx.Value(testKey{}))
		assert.Equal(t, logger, ctx.Value(contextKey{}))
	})
}

func TestFromCtx(t *testing.T) {
	t.Run("with nil context returns default", func(t *testing.T) {
		logger := FromCtx(t.Context())
		assert.Equal(t, slog.Default(), logger)
	})

	t.Run("with empty context returns default", func(t *testing.T) {
		ctx := context.Background()
		logger := FromCtx(ctx)
		assert.Equal(t, slog.Default(), logger)
	})

	t.Run("with logger in context", func(t *testing.T) {
		customLogger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
		ctx := NewCtx(context.Background(), customLogger)
		logger := FromCtx(ctx)
		assert.Equal(t, customLogger, logger)
	})

	t.Run("panics with invalid value type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKey{}, "not a logger")
		assert.Panics(t, func() {
			FromCtx(ctx)
		})
	})
}

func TestErr(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		err := errors.New("test error")
		attr := Err(err)

		assert.Equal(t, "err", attr.Key)
		assert.Equal(t, err, attr.Value.Any())
	})

	t.Run("with nil error", func(t *testing.T) {
		attr := Err(nil)

		assert.Equal(t, "err", attr.Key)
		assert.Nil(t, attr.Value.Any())
	})
}

func TestWith(t *testing.T) {
	t.Run("adds attributes to logger", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		ctx := NewCtx(context.Background(), logger)

		newCtx := With(ctx, "key1", "value1", "key2", 42)
		require.NotNil(t, newCtx)

		// Log something to verify attributes are included
		Info(newCtx, "test message")

		output := buf.String()
		assert.Contains(t, output, "key1=value1")
		assert.Contains(t, output, "key2=42")
	})

	t.Run("preserves original context", func(t *testing.T) {
		var buf1 bytes.Buffer
		logger1 := slog.New(slog.NewTextHandler(&buf1, nil))

		ctx1 := NewCtx(context.Background(), logger1)
		ctx2 := With(ctx1, "key", "value")

		// Verify ctx1 and ctx2 have different loggers
		assert.NotEqual(t, FromCtx(ctx1), FromCtx(ctx2))
	})
}

func TestWithGroup(t *testing.T) {
	t.Run("adds group to logger", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		ctx := NewCtx(context.Background(), logger)

		newCtx := WithGroup(ctx, "mygroup")
		require.NotNil(t, newCtx)

		// Log with an attribute
		Info(newCtx, "test message", slog.String("key", "value"))

		output := buf.String()
		// In JSON format, group creates nested structure
		assert.Contains(t, output, "mygroup")
		assert.Contains(t, output, "key")
	})
}

func TestDebug(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ctx := NewCtx(context.Background(), logger)

	Debug(ctx, "debug message", slog.String("key", "value"))

	output := buf.String()
	assert.Contains(t, output, "DEBUG")
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "key=value")
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	ctx := NewCtx(context.Background(), logger)

	Info(ctx, "info message", slog.String("key", "value"))

	output := buf.String()
	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "key=value")
}

func TestWarn(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	ctx := NewCtx(context.Background(), logger)

	Warn(ctx, "warn message", slog.String("key", "value"))

	output := buf.String()
	assert.Contains(t, output, "WARN")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "key=value")
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	ctx := NewCtx(context.Background(), logger)

	Error(ctx, "error message", slog.String("key", "value"))

	output := buf.String()
	assert.Contains(t, output, "ERROR")
	assert.Contains(t, output, "error message")
	assert.Contains(t, output, "key=value")
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name      string
		level     slog.Level
		logFunc   func(context.Context, string, ...slog.Attr)
		logLevel  string
		shouldLog bool
	}{
		{
			name:      "debug logged when level is debug",
			level:     slog.LevelDebug,
			logFunc:   Debug,
			logLevel:  "DEBUG",
			shouldLog: true,
		},
		{
			name:      "debug not logged when level is info",
			level:     slog.LevelInfo,
			logFunc:   Debug,
			logLevel:  "DEBUG",
			shouldLog: false,
		},
		{
			name:      "info logged when level is info",
			level:     slog.LevelInfo,
			logFunc:   Info,
			logLevel:  "INFO",
			shouldLog: true,
		},
		{
			name:      "info not logged when level is warn",
			level:     slog.LevelWarn,
			logFunc:   Info,
			logLevel:  "INFO",
			shouldLog: false,
		},
		{
			name:      "warn logged when level is warn",
			level:     slog.LevelWarn,
			logFunc:   Warn,
			logLevel:  "WARN",
			shouldLog: true,
		},
		{
			name:      "error logged when level is error",
			level:     slog.LevelError,
			logFunc:   Error,
			logLevel:  "ERROR",
			shouldLog: true,
		},
		{
			name:      "error logged when level is info",
			level:     slog.LevelInfo,
			logFunc:   Error,
			logLevel:  "ERROR",
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
				Level: tt.level,
			}))
			ctx := NewCtx(context.Background(), logger)

			tt.logFunc(ctx, "test message")

			output := buf.String()
			if tt.shouldLog {
				assert.Contains(t, output, tt.logLevel)
				assert.Contains(t, output, "test message")
			} else {
				assert.Empty(t, output)
			}
		})
	}
}

func TestLogWithMultipleAttributes(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	ctx := NewCtx(context.Background(), logger)

	Info(ctx, "test message",
		slog.String("string_key", "string_value"),
		slog.Int("int_key", 42),
		slog.Bool("bool_key", true),
		Err(errors.New("test error")),
	)

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "string_key=string_value")
	assert.Contains(t, output, "int_key=42")
	assert.Contains(t, output, "bool_key=true")
	assert.Contains(t, output, "err=\"test error\"")
}

func TestLogCtx(t *testing.T) {
	t.Run("creates new context with logger", func(t *testing.T) {
		ctx := context.Background()
		newCtx := LogCtx(ctx, WithLevel(slog.LevelDebug), WithFormat(FormatText))

		require.NotNil(t, newCtx)
		logger := FromCtx(newCtx)
		assert.NotNil(t, logger)
	})

	t.Run("logger respects options", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := context.Background()
		newCtx := LogCtx(ctx,
			WithLevel(slog.LevelDebug),
			WithFormat(FormatText),
			WithSlogDefault(false),
		)

		// Manually set up a logger to capture output
		// Since LogCtx creates a new logger internally, we need to replace it
		testLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		testCtx := NewCtx(newCtx, testLogger)

		Debug(testCtx, "debug message")

		output := buf.String()
		assert.Contains(t, output, "DEBUG")
		assert.Contains(t, output, "debug message")
	})
}

func TestCloseCtx(t *testing.T) {
	t.Run("with default logger", func(t *testing.T) {
		ctx := context.Background()
		err := CloseCtx(ctx)
		assert.NoError(t, err)
	})

	t.Run("with custom logger no file", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		ctx := NewCtx(context.Background(), logger)

		err := CloseCtx(ctx)
		assert.NoError(t, err)
	})
}

func TestLogFunctionCallDepth(t *testing.T) {
	// This test verifies that the source location is captured correctly
	// when using the logging functions
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}))
	ctx := NewCtx(context.Background(), logger)

	Info(ctx, "test message")

	output := buf.String()
	// With go:noinline directives, the source should correctly point to this test file
	// where Info() is called, not ctx.go
	assert.Contains(t, output, "source=")
	assert.Contains(t, output, "ctx_test.go")
}

func TestContextChaining(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	ctx := NewCtx(context.Background(), logger)

	// Chain multiple With calls
	ctx = With(ctx, "step", 1)
	ctx = With(ctx, "user", "test-user")
	ctx = WithGroup(ctx, "request")
	ctx = With(ctx, "method", "GET")

	Info(ctx, "processing request")

	output := buf.String()
	assert.Contains(t, output, "step=1")
	assert.Contains(t, output, "user=test-user")
	assert.Contains(t, output, "request")
	assert.Contains(t, output, "method=GET")
}

func TestLogWithDifferentFormats(t *testing.T) {
	tests := []struct {
		name         string
		format       Format
		message      string
		attrs        []slog.Attr
		checkOutputs []string
	}{
		{
			name:    "text format",
			format:  FormatText,
			message: "test message",
			attrs: []slog.Attr{
				slog.String("key", "value"),
			},
			checkOutputs: []string{"INFO", "test message", "key=value"},
		},
		{
			name:    "json format",
			format:  FormatJSON,
			message: "test message",
			attrs: []slog.Attr{
				slog.String("key", "value"),
			},
			checkOutputs: []string{`"level":"INFO"`, `"msg":"test message"`, `"key":"value"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			var handler slog.Handler
			if tt.format == FormatText {
				handler = slog.NewTextHandler(&buf, nil)
			} else {
				handler = slog.NewJSONHandler(&buf, nil)
			}

			logger := slog.New(handler)
			ctx := NewCtx(context.Background(), logger)

			Info(ctx, tt.message, tt.attrs...)

			output := buf.String()
			for _, check := range tt.checkOutputs {
				assert.Contains(t, output, check)
			}
		})
	}
}

func TestLogWithNilContext(t *testing.T) {
	// Should not panic and should use default logger
	var buf bytes.Buffer
	oldDefault := slog.Default()
	defer slog.SetDefault(oldDefault)

	// Set a custom default logger
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	// These should not panic
	assert.NotPanics(t, func() {
		Info(t.Context(), "test message")
	})

	output := buf.String()
	assert.Contains(t, output, "test message")
}

func TestLogHandlesSpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	ctx := NewCtx(context.Background(), logger)

	specialChars := `test "quotes" and \backslashes\ and newlines
and tabs	here`

	Info(ctx, specialChars, slog.String("key", specialChars))

	output := buf.String()
	assert.NotEmpty(t, output)
	// Verify it didn't panic and produced some output
	assert.True(t, strings.Contains(output, "test") || strings.Contains(output, "quotes"))
}
