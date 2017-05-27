// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package zap

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap/zapcore"
)

// A Logger provides fast, leveled, structured logging. All methods are safe for
// concurrent use.
//
// If you'd prefer a slower but less verbose logger, see the SugaredLogger.
type Logger struct {
	core zapcore.Core

	dpanicAction zapcore.CheckWriteAction
	name         string
	errorOutput  zapcore.WriteSyncer

	addCaller bool
	addStack  zapcore.LevelEnabler

	callerSkip int
}

// New constructs a new Logger from the provided zapcore.Core and Options. If
// the passed zapcore.Core is nil, we fall back to using a no-op
// implementation.
//
// This is the most flexible way to construct a Logger, but also the most
// verbose. For typical use cases, NewProduction and NewDevelopment are more
// convenient.
func New(core zapcore.Core, options ...Option) *Logger {
	if core == nil {
		return NewNop()
	}
	log := &Logger{
		core:        core,
		errorOutput: zapcore.Lock(os.Stderr),
		addStack:    zapcore.FatalLevel + 1,
	}
	return log.WithOptions(options...)
}

// NewNop returns a no-op Logger. It never writes out logs or internal errors,
// and it never runs user-defined hooks.
//
// Using WithOptions to replace the Core or error output of a no-op Logger can
// re-enable logging.
func NewNop() *Logger {
	return &Logger{
		core:        zapcore.NewNopCore(),
		errorOutput: zapcore.AddSync(ioutil.Discard),
		addStack:    zapcore.FatalLevel + 1,
	}
}

// NewProduction builds a sensible production Logger that writes InfoLevel and
// above logs to standard error as JSON.
//
// It's a shortcut for NewProductionConfig().Build(...Option).
func NewProduction(options ...Option) (*Logger, error) {
	return NewProductionConfig().Build(options...)
}

// NewDevelopment builds a development Logger that writes DebugLevel and above
// logs to standard error in a human-friendly format.
//
// It's a shortcut for NewDevelopmentConfig().Build(...Option).
func NewDevelopment(options ...Option) (*Logger, error) {
	return NewDevelopmentConfig().Build(options...)
}

// Sugar converts a Logger to a SugaredLogger.
func (log *Logger) Sugar() *SugaredLogger {
	core := log.clone()
	core.callerSkip += 2
	return &SugaredLogger{core}
}

// Named adds a new path segment to the logger's name. Segments are joined by
// periods.
func (log *Logger) Named(s string) *Logger {
	if s == "" {
		return log
	}
	l := log.clone()
	if log.name == "" {
		l.name = s
	} else {
		l.name = strings.Join([]string{l.name, s}, ".")
	}
	return l
}

// WithOptions clones the current Logger, applies the supplied Options, and
// returns the resulting Logger. It's safe to use concurrently.
func (log *Logger) WithOptions(opts ...Option) *Logger {
	c := log.clone()
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}

// With creates a child logger and adds structured context to it. Fields added
// to the child don't affect the parent, and vice versa.
func (log *Logger) With(fields ...zapcore.Field) *Logger {
	if len(fields) == 0 {
		return log
	}
	l := log.clone()
	l.core = l.core.With(fields)
	return l
}

// Check returns a CheckedEntry if logging a message at the specified level
// is enabled. It's a completely optional optimization; in high-performance
// applications, Check can help avoid allocating a slice to hold fields.
func (log *Logger) Check(lvl zapcore.Level, msg string) *zapcore.CheckedEntry {
	var action zapcore.CheckWriteAction

	switch lvl {
	case PanicLevel:
		action = zapcore.WriteThenPanic
	case FatalLevel:
		action = zapcore.WriteThenFatal
	case DPanicLevel:
		action = log.dpanicAction
	default:
		action = zapcore.WriteThenNoop
	}

	return log.CheckWithAction(lvl, action, msg)
}

// CheckWithAction similarly to Check returns a CheckedEntry if logging a
// message at the specified level is enabled. But additionally it allows
// one to specify the action to take after writing out the message.
func (log *Logger) CheckWithAction(lvl zapcore.Level, action zapcore.CheckWriteAction, msg string) *zapcore.CheckedEntry {
	// check must always be called directly by a method in the Logger interface
	// (e.g., Check, Info, Fatal).
	const callerSkipOffset = 2

	// Create basic checked entry thru the core; this will be non-nil if the
	// log message will actually be written somewhere.
	ent := zapcore.Entry{
		LoggerName: log.name,
		Time:       time.Now(),
		Level:      lvl,
		Message:    msg,
	}
	ce := log.core.Check(ent, nil)
	willWrite := ce != nil

	// Set up any required terminal behavior.
	if action != zapcore.WriteThenNoop {
		ce = ce.Should(ent, action)
	}

	// Only do further annotation if we're going to write this message; checked
	// entries that exist only for terminal behavior don't benefit from
	// annotation.
	if !willWrite {
		return ce
	}

	// Thread the error output through to the CheckedEntry.
	ce.ErrorOutput = log.errorOutput
	if log.addCaller {
		ce.Entry.Caller = zapcore.NewEntryCaller(runtime.Caller(log.callerSkip + callerSkipOffset))
		if !ce.Entry.Caller.Defined {
			fmt.Fprintf(log.errorOutput, "%v Logger.check error: failed to get caller\n", time.Now().UTC())
			log.errorOutput.Sync()
		}
	}
	if log.addStack.Enabled(ce.Entry.Level) {
		ce.Entry.Stack = Stack("").String
	}

	return ce
}

// Debug logs a message at DebugLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log *Logger) Debug(msg string, fields ...zapcore.Field) {
	if ce := log.CheckWithAction(DebugLevel, zapcore.WriteThenNoop, msg); ce != nil {
		ce.Write(fields...)
	}
}

// Info logs a message at InfoLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log *Logger) Info(msg string, fields ...zapcore.Field) {
	if ce := log.CheckWithAction(InfoLevel, zapcore.WriteThenNoop, msg); ce != nil {
		ce.Write(fields...)
	}
}

// Warn logs a message at WarnLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log *Logger) Warn(msg string, fields ...zapcore.Field) {
	if ce := log.CheckWithAction(WarnLevel, zapcore.WriteThenNoop, msg); ce != nil {
		ce.Write(fields...)
	}
}

// Error logs a message at ErrorLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log *Logger) Error(msg string, fields ...zapcore.Field) {
	if ce := log.CheckWithAction(ErrorLevel, zapcore.WriteThenNoop, msg); ce != nil {
		ce.Write(fields...)
	}
}

// DPanic logs a message at DPanicLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// If the logger is in development mode, it then panics (DPanic means
// "development panic"). This is useful for catching errors that are
// recoverable, but shouldn't ever happen.
func (log *Logger) DPanic(msg string, fields ...zapcore.Field) {
	if ce := log.CheckWithAction(DPanicLevel, log.dpanicAction, msg); ce != nil {
		ce.Write(fields...)
	}
}

// Panic logs a message at PanicLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// The logger then panics, even if logging at PanicLevel is disabled.
func (log *Logger) Panic(msg string, fields ...zapcore.Field) {
	if ce := log.CheckWithAction(PanicLevel, zapcore.WriteThenPanic, msg); ce != nil {
		ce.Write(fields...)
	}
}

// Fatal logs a message at FatalLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// The logger then calls os.Exit(1), even if logging at FatalLevel is disabled.
func (log *Logger) Fatal(msg string, fields ...zapcore.Field) {
	if ce := log.CheckWithAction(FatalLevel, zapcore.WriteThenFatal, msg); ce != nil {
		ce.Write(fields...)
	}
}

// Sync flushes any buffered log entries.
func (log *Logger) Sync() error {
	return log.core.Sync()
}

// Core returns the underlying zapcore.Core.
func (log *Logger) Core() zapcore.Core {
	return log.core
}

func (log *Logger) clone() *Logger {
	copy := *log
	return &copy
}
