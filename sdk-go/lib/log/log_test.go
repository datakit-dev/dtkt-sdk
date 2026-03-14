package log

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	t.Run("creates logger with default config", func(t *testing.T) {
		// Save and restore original default
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		// Clear environment variables
		//nolint:errcheck
		os.Unsetenv("DTKT_APP_ENV")
		//nolint:errcheck
		os.Unsetenv("DTKT_LOG_LEVEL")
		//nolint:errcheck
		os.Unsetenv("DTKT_LOG_FORMAT")

		logger := NewLogger()
		require.NotNil(t, logger)
	})

	t.Run("creates logger with custom level", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		logger := NewLogger(WithLevel(slog.LevelDebug), WithSlogDefault(false))
		require.NotNil(t, logger)

		// Verify the logger respects the level by checking if debug is enabled
		ctx := NewCtx(t.Context(), logger)
		assert.True(t, logger.Enabled(ctx, slog.LevelDebug))
	})

	t.Run("creates logger with text format", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		logger := NewLogger(WithFormat(FormatText), WithSlogDefault(false))
		require.NotNil(t, logger)
	})

	t.Run("creates logger with json format", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		logger := NewLogger(WithFormat(FormatJSON), WithSlogDefault(false))
		require.NotNil(t, logger)
	})

	t.Run("creates logger with source enabled", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		logger := NewLogger(WithSource(true), WithSlogDefault(false))
		require.NotNil(t, logger)
	})

	t.Run("sets as default logger when configured", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		logger := NewLogger(WithSlogDefault(true))
		require.NotNil(t, logger)

		// Verify it was set as default
		assert.Equal(t, logger, slog.Default())
	})

	t.Run("does not set as default when configured", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		logger := NewLogger(WithSlogDefault(false))
		require.NotNil(t, logger)

		// Verify it was NOT set as default
		assert.NotEqual(t, logger, slog.Default())
		assert.Equal(t, oldDefault, slog.Default())
	})

	t.Run("handles nil options", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		logger := NewLogger(nil, WithSlogDefault(false), nil)
		require.NotNil(t, logger)
	})

	t.Run("applies multiple options", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		logger := NewLogger(
			WithLevel(slog.LevelWarn),
			WithFormat(FormatJSON),
			WithSource(false),
			WithSlogDefault(false),
		)
		require.NotNil(t, logger)

		ctx := NewCtx(t.Context(), logger)
		assert.True(t, logger.Enabled(ctx, slog.LevelWarn))
		assert.True(t, logger.Enabled(ctx, slog.LevelError))
		assert.False(t, logger.Enabled(ctx, slog.LevelInfo))
		assert.False(t, logger.Enabled(ctx, slog.LevelDebug))
	})
}

func TestNewLoggerWithFile(t *testing.T) {
	t.Run("creates file logger", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		// Create temp directory
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "test.log")

		logger := NewLogger(WithFile(logFile), WithSlogDefault(false))
		require.NotNil(t, logger)

		// Write a log message
		ctx := NewCtx(t.Context(), logger)
		Info(ctx, "test message", slog.String("key", "value"))

		// Close the logger to flush
		err := CloseLogger(logger)
		require.NoError(t, err)

		// Verify file was created and contains the message
		content, err := os.ReadFile(logFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test message")
	})

	t.Run("creates nested directories for log file", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "nested", "path", "test.log")

		logger := NewLogger(WithFile(logFile), WithSlogDefault(false))
		require.NotNil(t, logger)

		// Write a log message
		ctx := NewCtx(t.Context(), logger)
		Info(ctx, "test message")

		// Close the logger
		err := CloseLogger(logger)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(logFile)
		assert.NoError(t, err)
	})

	t.Run("appends to existing log file", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "test.log")

		// Create first logger and write
		logger1 := NewLogger(WithFile(logFile), WithSlogDefault(false))
		ctx1 := NewCtx(t.Context(), logger1)
		Info(ctx1, "first message")
		err := CloseLogger(logger1)
		require.NoError(t, err)

		// Create second logger and write
		logger2 := NewLogger(WithFile(logFile), WithSlogDefault(false))
		ctx2 := NewCtx(t.Context(), logger2)
		Info(ctx2, "second message")
		err = CloseLogger(logger2)
		require.NoError(t, err)

		// Verify both messages are in file
		content, err := os.ReadFile(logFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "first message")
		assert.Contains(t, string(content), "second message")
	})

	t.Run("file has correct permissions", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "test.log")

		logger := NewLogger(WithFile(logFile), WithSlogDefault(false))
		require.NotNil(t, logger)

		ctx := NewCtx(t.Context(), logger)
		Info(ctx, "test message")

		err := CloseLogger(logger)
		require.NoError(t, err)

		// Check file permissions
		fileInfo, err := os.Stat(logFile)
		require.NoError(t, err)

		// File should be created with 0600 permissions
		mode := fileInfo.Mode().Perm()
		assert.Equal(t, os.FileMode(0o600), mode)
	})

	t.Run("panics when directory cannot be created", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		// Try to create a log file in a path that cannot be created
		// Use a path that's likely to fail (e.g., under a file instead of directory)
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "file.txt")
		err := os.WriteFile(tmpFile, []byte("test"), 0o600)
		require.NoError(t, err)

		logFile := filepath.Join(tmpFile, "nested", "test.log")

		assert.Panics(t, func() {
			NewLogger(WithFile(logFile), WithSlogDefault(false))
		})
	})
}

func TestCloseLogger(t *testing.T) {
	t.Run("close logger without file", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		logger := NewLogger(WithSlogDefault(false))
		err := CloseLogger(logger)
		assert.NoError(t, err)
	})

	t.Run("close logger with file", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "test.log")

		logger := NewLogger(WithFile(logFile), WithSlogDefault(false))
		ctx := NewCtx(t.Context(), logger)
		Info(ctx, "test message")

		err := CloseLogger(logger)
		assert.NoError(t, err)

		// Verify the file handle is removed from map
		_, exists := logFiles[logger]
		assert.False(t, exists)
	})

	t.Run("close logger removes from map", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "test.log")

		logger := NewLogger(WithFile(logFile), WithSlogDefault(false))

		// Verify it's in the map
		_, exists := logFiles[logger]
		assert.True(t, exists)

		// Close and verify it's removed
		err := CloseLogger(logger)
		assert.NoError(t, err)

		_, exists = logFiles[logger]
		assert.False(t, exists)
	})

	t.Run("close same logger twice", func(t *testing.T) {
		oldDefault := slog.Default()
		defer slog.SetDefault(oldDefault)

		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "test.log")

		logger := NewLogger(WithFile(logFile), WithSlogDefault(false))

		err := CloseLogger(logger)
		assert.NoError(t, err)

		// Closing again should not error
		err = CloseLogger(logger)
		assert.NoError(t, err)
	})
}

func TestLoggerOutputFormat(t *testing.T) {
	tests := []struct {
		name           string
		format         Format
		level          slog.Level
		message        string
		attrs          []slog.Attr
		containsChecks []string
	}{
		{
			name:    "text format info level",
			format:  FormatText,
			level:   slog.LevelInfo,
			message: "test message",
			attrs: []slog.Attr{
				slog.String("key", "value"),
			},
			containsChecks: []string{"INFO", "test message", "key=value"},
		},
		{
			name:    "json format info level",
			format:  FormatJSON,
			level:   slog.LevelInfo,
			message: "test message",
			attrs: []slog.Attr{
				slog.String("key", "value"),
			},
			containsChecks: []string{`"level":"INFO"`, `"msg":"test message"`, `"key":"value"`},
		},
		{
			name:    "text format debug level",
			format:  FormatText,
			level:   slog.LevelDebug,
			message: "debug message",
			attrs: []slog.Attr{
				slog.Int("count", 42),
			},
			containsChecks: []string{"DEBUG", "debug message", "count=42"},
		},
		{
			name:    "json format error level",
			format:  FormatJSON,
			level:   slog.LevelError,
			message: "error message",
			attrs: []slog.Attr{
				Err(assert.AnError),
			},
			containsChecks: []string{`"level":"ERROR"`, `"msg":"error message"`, `"err"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldDefault := slog.Default()
			defer slog.SetDefault(oldDefault)

			tmpDir := t.TempDir()
			logFile := filepath.Join(tmpDir, "test.log")

			logger := NewLogger(
				WithFile(logFile),
				WithFormat(tt.format),
				WithLevel(slog.LevelDebug),
				WithSlogDefault(false),
			)

			ctx := NewCtx(t.Context(), logger)

			// Log the message
			switch tt.level {
			case slog.LevelDebug:
				Debug(ctx, tt.message, tt.attrs...)
			case slog.LevelInfo:
				Info(ctx, tt.message, tt.attrs...)
			case slog.LevelWarn:
				Warn(ctx, tt.message, tt.attrs...)
			case slog.LevelError:
				Error(ctx, tt.message, tt.attrs...)
			}

			err := CloseLogger(logger)
			require.NoError(t, err)

			// Read and verify
			content, err := os.ReadFile(logFile)
			require.NoError(t, err)

			output := string(content)
			for _, check := range tt.containsChecks {
				assert.Contains(t, output, check)
			}
		})
	}
}

func TestLoggerWithSource(t *testing.T) {
	tests := []struct {
		name          string
		includeSource bool
		shouldContain bool
	}{
		{
			name:          "with source enabled",
			includeSource: true,
			shouldContain: true,
		},
		{
			name:          "with source disabled",
			includeSource: false,
			shouldContain: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldDefault := slog.Default()
			defer slog.SetDefault(oldDefault)

			tmpDir := t.TempDir()
			logFile := filepath.Join(tmpDir, "test.log")

			logger := NewLogger(
				WithFile(logFile),
				WithFormat(FormatJSON),
				WithSource(tt.includeSource),
				WithSlogDefault(false),
			)

			ctx := NewCtx(t.Context(), logger)
			Info(ctx, "test message") // Source should point to this line

			err := CloseLogger(logger)
			require.NoError(t, err)

			content, err := os.ReadFile(logFile)
			require.NoError(t, err)

			output := string(content)

			if tt.shouldContain {
				// Verify source field is present with actual file and line information
				// Source should point to where Info() was called from (this test file)
				assert.Contains(t, output, `"source"`)
				assert.Contains(t, output, "log_test.go")
				// Verify it contains a valid line number in the source structure
				assert.Regexp(t, `"source":\{"function":"[^"]+","file":"[^"]+log_test\.go","line":\d+\}`, output)
			} else {
				assert.NotContains(t, output, `"source"`)
			}
		})
	}
}

func TestMultipleLoggersToSameFile(t *testing.T) {
	oldDefault := slog.Default()
	defer slog.SetDefault(oldDefault)

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger1 := NewLogger(WithFile(logFile), WithSlogDefault(false))
	logger2 := NewLogger(WithFile(logFile), WithSlogDefault(false))

	ctx1 := NewCtx(t.Context(), logger1)
	ctx2 := NewCtx(t.Context(), logger2)

	Info(ctx1, "message from logger1")
	Info(ctx2, "message from logger2")

	err := CloseLogger(logger1)
	require.NoError(t, err)

	err = CloseLogger(logger2)
	require.NoError(t, err)

	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	output := string(content)
	assert.Contains(t, output, "message from logger1")
	assert.Contains(t, output, "message from logger2")
}

func TestFormatConstants(t *testing.T) {
	assert.Equal(t, Format("text"), FormatText)
	assert.Equal(t, Format("json"), FormatJSON)
}

func TestLoggerIntegration(t *testing.T) {
	// Integration test that verifies the entire flow
	oldDefault := slog.Default()
	defer slog.SetDefault(oldDefault)

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "integration.log")

	// Create logger with various options
	logger := NewLogger(
		WithFile(logFile),
		WithFormat(FormatJSON),
		WithLevel(slog.LevelDebug),
		WithSource(true),
		WithSlogDefault(false),
	)

	// Create context and add attributes
	ctx := NewCtx(t.Context(), logger)
	ctx = With(ctx, "request_id", "12345")
	ctx = WithGroup(ctx, "user")
	ctx = With(ctx, "id", "user-001", "name", "Test User")

	// Log at various levels
	Debug(ctx, "debug message")
	Info(ctx, "info message")
	Warn(ctx, "warn message")
	Error(ctx, "error message", Err(assert.AnError))

	// Close logger
	err := CloseLogger(logger)
	require.NoError(t, err)

	// Verify output
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	output := string(content)

	// Verify all levels were logged
	assert.Contains(t, output, `"level":"DEBUG"`)
	assert.Contains(t, output, `"level":"INFO"`)
	assert.Contains(t, output, `"level":"WARN"`)
	assert.Contains(t, output, `"level":"ERROR"`)

	// Verify messages
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")

	// Verify attributes
	assert.Contains(t, output, "request_id")
	assert.Contains(t, output, "12345")

	// Verify source is included
	assert.Contains(t, output, "source")

	// Verify we have multiple log lines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.GreaterOrEqual(t, len(lines), 4)
}

func TestLoggerConcurrentWrites(t *testing.T) {
	oldDefault := slog.Default()
	defer slog.SetDefault(oldDefault)

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "concurrent.log")

	logger := NewLogger(
		WithFile(logFile),
		WithFormat(FormatJSON),
		WithSlogDefault(false),
	)
	//nolint:errcheck
	defer CloseLogger(logger)

	ctx := NewCtx(t.Context(), logger)

	// Write from multiple goroutines
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				Info(ctx, "concurrent message", slog.Int("goroutine", id), slog.Int("iteration", j))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	err := CloseLogger(logger)
	require.NoError(t, err)

	// Verify file has content
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	output := string(content)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have 100 log lines (10 goroutines * 10 iterations)
	assert.Equal(t, 100, len(lines))
}
