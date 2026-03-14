package log

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"
)

type contextKey struct{}

func NewCtx(ctx context.Context, logger *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, contextKey{}, logger)
}

func FromCtx(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}

	v := ctx.Value(contextKey{})

	if v == nil {
		return slog.Default()
	}

	if logger, ok := v.(*slog.Logger); ok {
		return logger
	}

	panic(fmt.Sprintf("unexpected value type for log context key: %T", v))
}

func Err(err error) slog.Attr {
	return slog.Any("err", err)
}

func With(ctx context.Context, args ...any) context.Context {
	return NewCtx(ctx, FromCtx(ctx).With(args...))
}

func WithGroup(ctx context.Context, name string) context.Context {
	return NewCtx(ctx, FromCtx(ctx).WithGroup(name))
}

//go:noinline
func Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	log(ctx, slog.LevelDebug, msg, attrs...)
}

//go:noinline
func Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	log(ctx, slog.LevelInfo, msg, attrs...)
}

//go:noinline
func Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	log(ctx, slog.LevelWarn, msg, attrs...)
}

//go:noinline
func Error(ctx context.Context, msg string, attrs ...slog.Attr) {
	log(ctx, slog.LevelError, msg, attrs...)
}

//go:noinline
func Fatal(ctx context.Context, msg string, attrs ...slog.Attr) {
	log(ctx, 12, msg, attrs...)
	os.Exit(1)
}

func CloseCtx(ctx context.Context) error {
	return CloseLogger(FromCtx(ctx))
}

func LogCtx(ctx context.Context, options ...Option) context.Context {
	return NewCtx(ctx, NewLogger(options...))
}

func log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	l := FromCtx(ctx)
	if !l.Enabled(ctx, level) {
		return
	}

	const callDepth = 2
	pc, _, _, _ := runtime.Caller(callDepth)

	r := slog.NewRecord(time.Now(), level, msg, pc)
	r.AddAttrs(attrs...)

	_ = l.Handler().Handle(ctx, r)
}
