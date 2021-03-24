package zap

import (
	"context"

	"go.uber.org/zap/zapcore"
)

// LoggerWithCtx is a wrapper for Logger that also carries a context.Context.
type LoggerWithCtx struct {
	v   *Logger
	ctx context.Context
}

func (log *Logger) Ctx(ctx context.Context) LoggerWithCtx {
	return LoggerWithCtx{
		ctx: ctx,
		v:   log,
	}
}

// Check returns a CheckedEntry if logging a message at the specified level
// is enabled. It's a completely optional optimization; in high-performance
// applications, Check can help avoid allocating a slice to hold fields.
func (log LoggerWithCtx) Check(lvl zapcore.Level, msg string) *zapcore.CheckedEntry {
	return log.v.check(log.ctx, lvl, msg)
}

// Debug logs a message at DebugLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log LoggerWithCtx) Debug(msg string, fields ...Field) {
	if ce := log.v.check(log.ctx, DebugLevel, msg); ce != nil {
		ce.Write(fields...)
	}
}

// Info logs a message at InfoLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log LoggerWithCtx) Info(msg string, fields ...Field) {
	if ce := log.v.check(log.ctx, InfoLevel, msg); ce != nil {
		ce.Write(fields...)
	}
}

// Warn logs a message at WarnLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log LoggerWithCtx) Warn(msg string, fields ...Field) {
	if ce := log.v.check(log.ctx, WarnLevel, msg); ce != nil {
		ce.Write(fields...)
	}
}

// Error logs a message at ErrorLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log LoggerWithCtx) Error(msg string, fields ...Field) {
	if ce := log.v.check(log.ctx, ErrorLevel, msg); ce != nil {
		ce.Write(fields...)
	}
}

// DPanic logs a message at DPanicLevel. The message includes any fields
// passed at the log site, as well as any fields accumulated on the logger.
//
// If the logger is in development mode, it then panics (DPanic means
// "development panic"). This is useful for catching errors that are
// recoverable, but shouldn't ever happen.
func (log LoggerWithCtx) DPanic(msg string, fields ...Field) {
	if ce := log.v.check(log.ctx, DPanicLevel, msg); ce != nil {
		ce.Write(fields...)
	}
}

// Panic logs a message at PanicLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// The logger then panics, even if logging at PanicLevel is disabled.
func (log LoggerWithCtx) Panic(msg string, fields ...Field) {
	if ce := log.v.check(log.ctx, PanicLevel, msg); ce != nil {
		ce.Write(fields...)
	}
}

// Fatal logs a message at FatalLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// The logger then calls os.Exit(1), even if logging at FatalLevel is
// disabled.
func (log LoggerWithCtx) Fatal(msg string, fields ...Field) {
	if ce := log.v.check(log.ctx, FatalLevel, msg); ce != nil {
		ce.Write(fields...)
	}
}

// Sync calls the underlying Core's Sync method, flushing any buffered log
// entries. Applications should take care to call Sync before exiting.
func (log LoggerWithCtx) Sync() error {
	return log.v.core.Sync()
}

// Core returns the Logger's underlying zapcore.Core.
func (log LoggerWithCtx) Core() zapcore.Core {
	return log.v.core
}

// WithOptions clones the current Logger, applies the supplied Options, and
// returns the resulting Logger. It's safe to use concurrently.
func (log LoggerWithCtx) WithOptions(opts ...Option) LoggerWithCtx {
	return LoggerWithCtx{
		v:   log.v.WithOptions(opts...),
		ctx: log.ctx,
	}
}
