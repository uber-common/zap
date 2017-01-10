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
	"sync"
	"testing"

	"go.uber.org/zap/internal/exit"
	"go.uber.org/zap/testutils"
	"go.uber.org/zap/zapcore"

	"github.com/stretchr/testify/assert"
)

func TestLoggerDynamicLevel(t *testing.T) {
	// Test that the DynamicLevel applys to all ancestors and descendants.
	dl := DynamicLevel()

	withLogger(t, dl, nil, func(grandparent Logger, _ *zapcore.ObservedLogs) {
		parent := grandparent.With(Int("generation", 1))
		child := parent.With(Int("generation", 2))

		tests := []struct {
			setLevel  zapcore.Level
			testLevel zapcore.Level
			enabled   bool
		}{
			{DebugLevel, DebugLevel, true},
			{InfoLevel, DebugLevel, false},
			{WarnLevel, PanicLevel, true},
		}

		for _, tt := range tests {
			dl.SetLevel(tt.setLevel)
			for _, logger := range []Logger{grandparent, parent, child} {
				if tt.enabled {
					assert.NotNil(
						t,
						logger.Check(tt.testLevel, ""),
						"Expected level %s to be enabled after setting level %s.", tt.testLevel, tt.setLevel,
					)
				} else {
					assert.Nil(
						t,
						logger.Check(tt.testLevel, ""),
						"Expected level %s to be enabled after setting level %s.", tt.testLevel, tt.setLevel,
					)
				}
			}
		}
	})
}

func TestLoggerInitialFields(t *testing.T) {
	fieldOpts := opts(Fields(Int("foo", 42), String("bar", "baz")))
	withLogger(t, DebugLevel, fieldOpts, func(logger Logger, logs *zapcore.ObservedLogs) {
		logger.Info("")
		assert.Equal(
			t,
			zapcore.ObservedLog{Context: []zapcore.Field{Int("foo", 42), String("bar", "baz")}},
			logs.AllUntimed()[0],
			"Unexpected output with initial fields set.",
		)
	})
}

func TestLoggerWith(t *testing.T) {
	fieldOpts := opts(Fields(Int("foo", 42)))
	withLogger(t, DebugLevel, fieldOpts, func(logger Logger, logs *zapcore.ObservedLogs) {
		// Child loggers should have copy-on-write semantics, so two children
		// shouldn't stomp on each other's fields or affect the parent's fields.
		logger.With(String("one", "two")).Info("")
		logger.With(String("three", "four")).Info("")
		logger.Info("")

		assert.Equal(t, []zapcore.ObservedLog{
			{Context: []zapcore.Field{Int("foo", 42), String("one", "two")}},
			{Context: []zapcore.Field{Int("foo", 42), String("three", "four")}},
			{Context: []zapcore.Field{Int("foo", 42)}},
		}, logs.AllUntimed(), "Unexpected cross-talk between child loggers.")
	})
}

func TestLoggerLogPanic(t *testing.T) {
	for _, tt := range []struct {
		do       func(Logger)
		should   bool
		expected string
	}{
		{func(logger Logger) { logger.Check(PanicLevel, "bar").Write() }, true, "bar"},
		{func(logger Logger) { logger.Panic("baz") }, true, "baz"},
	} {
		withLogger(t, DebugLevel, nil, func(logger Logger, logs *zapcore.ObservedLogs) {
			if tt.should {
				assert.Panics(t, func() { tt.do(logger) }, "Expected panic")
			} else {
				assert.NotPanics(t, func() { tt.do(logger) }, "Expected no panic")
			}

			output := logs.AllUntimed()
			assert.Equal(t, 1, len(output), "Unexpected number of logs.")
			assert.Equal(t, 0, len(output[0].Context), "Unexpected context on first log.")
			assert.Equal(
				t,
				zapcore.Entry{Message: tt.expected, Level: PanicLevel},
				output[0].Entry,
				"Unexpected output from panic-level Log.",
			)
		})
	}
}

func TestLoggerLogFatal(t *testing.T) {
	for _, tt := range []struct {
		do       func(Logger)
		expected string
	}{
		{func(logger Logger) { logger.Check(FatalLevel, "bar").Write() }, "bar"},
		{func(logger Logger) { logger.Fatal("baz") }, "baz"},
	} {
		withLogger(t, DebugLevel, nil, func(logger Logger, logs *zapcore.ObservedLogs) {
			stub := exit.WithStub(func() {
				tt.do(logger)
			})
			assert.True(t, stub.Exited, "Expected Fatal logger call to terminate process.")
			output := logs.AllUntimed()
			assert.Equal(t, 1, len(output), "Unexpected number of logs.")
			assert.Equal(t, 0, len(output[0].Context), "Unexpected context on first log.")
			assert.Equal(
				t,
				zapcore.Entry{Message: tt.expected, Level: FatalLevel},
				output[0].Entry,
				"Unexpected output from fatal-level Log.",
			)
		})
	}
}

func TestLoggerLeveledMethods(t *testing.T) {
	withLogger(t, DebugLevel, nil, func(logger Logger, logs *zapcore.ObservedLogs) {
		tests := []struct {
			method        func(string, ...zapcore.Field)
			expectedLevel zapcore.Level
		}{
			{logger.Debug, DebugLevel},
			{logger.Info, InfoLevel},
			{logger.Warn, WarnLevel},
			{logger.Error, ErrorLevel},
			{logger.DPanic, DPanicLevel},
		}
		for i, tt := range tests {
			tt.method("")
			output := logs.AllUntimed()
			assert.Equal(t, i+1, len(output), "Unexpected number of logs.")
			assert.Equal(t, 0, len(output[i].Context), "Unexpected context on first log.")
			assert.Equal(
				t,
				zapcore.Entry{Level: tt.expectedLevel},
				output[i].Entry,
				"Unexpected output from %s-level logger method.", tt.expectedLevel)
		}
	})
}

func TestLoggerAlwaysPanics(t *testing.T) {
	// Users can disable writing out panic-level logs, but calls to logger.Panic()
	// should still call panic().
	withLogger(t, FatalLevel, nil, func(logger Logger, logs *zapcore.ObservedLogs) {
		msg := "Even if output is disabled, logger.Panic should always panic."
		assert.Panics(t, func() { logger.Panic("foo") }, msg)
		assert.Panics(t, func() {
			if ce := logger.Check(PanicLevel, "foo"); ce != nil {
				ce.Write()
			}
		}, msg)
		assert.Equal(t, 0, logs.Len(), "Panics shouldn't be written out if PanicLevel is disabled.")
	})
}

func TestLoggerAlwaysFatals(t *testing.T) {
	// Users can disable writing out fatal-level logs, but calls to logger.Fatal()
	// should still terminate the process.
	withLogger(t, FatalLevel+1, nil, func(logger Logger, logs *zapcore.ObservedLogs) {
		stub := exit.WithStub(func() { logger.Fatal("") })
		assert.True(t, stub.Exited, "Expected calls to logger.Fatal to terminate process.")

		stub = exit.WithStub(func() {
			if ce := logger.Check(FatalLevel, ""); ce != nil {
				ce.Write()
			}
		})
		assert.True(t, stub.Exited, "Expected calls to logger.Check(FatalLevel, ...) to terminate process.")

		assert.Equal(t, 0, logs.Len(), "Shouldn't write out logs when fatal-level logging is disabled.")
	})
}

func TestLoggerDPanic(t *testing.T) {
	withLogger(t, DebugLevel, nil, func(logger Logger, logs *zapcore.ObservedLogs) {
		assert.NotPanics(t, func() { logger.DPanic("") })
		assert.Equal(
			t,
			[]zapcore.ObservedLog{{Entry: zapcore.Entry{Level: DPanicLevel}, Context: []zapcore.Field{}}},
			logs.AllUntimed(),
			"Unexpected log output from DPanic in production mode.",
		)
	})
	withLogger(t, DebugLevel, opts(Development()), func(logger Logger, logs *zapcore.ObservedLogs) {
		assert.Panics(t, func() { logger.DPanic("") })
		assert.Equal(
			t,
			[]zapcore.ObservedLog{{Entry: zapcore.Entry{Level: DPanicLevel}, Context: []zapcore.Field{}}},
			logs.AllUntimed(),
			"Unexpected log output from DPanic in development mode.",
		)
	})
}

func TestLoggerNoOpsDisabledLevels(t *testing.T) {
	withLogger(t, WarnLevel, nil, func(logger Logger, logs *zapcore.ObservedLogs) {
		logger.Info("silence!")
		assert.Equal(
			t,
			[]zapcore.ObservedLog{},
			logs.AllUntimed(),
			"Expected logging at a disabled level to produce no output.",
		)
	})
}

func TestLoggerWriteEntryFailure(t *testing.T) {
	errSink := &testutils.Buffer{}
	logger := New(
		zapcore.WriterFacility(
			zapcore.NewJSONEncoder(defaultEncoderConfig()),
			zapcore.Lock(zapcore.AddSync(testutils.FailWriter{})),
			DebugLevel,
		),
		ErrorOutput(errSink),
	)

	logger.Info("foo")
	// Should log the error.
	assert.Regexp(t, `write error: failed`, errSink.Stripped(), "Expected to log the error to the error output.")
	assert.True(t, errSink.Called(), "Expected logging an internal error to call Sync the error sink.")
}

func TestLoggerAddCaller(t *testing.T) {
	tests := []struct {
		options []Option
		pat     string
	}{
		{opts(AddCaller()), `.+/logger_test.go:[\d]+$`},
		{opts(AddCaller(), AddCallerSkip(1), AddCallerSkip(-1)), `.+/zap/logger_test.go:[\d]+$`},
		{opts(AddCaller(), AddCallerSkip(1)), `.+/zap/common_test.go:[\d]+$`},
		{opts(AddCaller(), AddCallerSkip(1), AddCallerSkip(3)), `.+/src/runtime/.*:[\d]+$`},
	}
	for _, tt := range tests {
		withLogger(t, DebugLevel, tt.options, func(logger Logger, logs *zapcore.ObservedLogs) {
			logger.Info("")
			output := logs.AllUntimed()
			assert.Equal(t, 1, len(output), "Unexpected number of logs written out.")
			assert.Regexp(
				t,
				tt.pat,
				output[0].Entry.Caller,
				"Expected to find package name and file name in output.",
			)
		})
	}
}

func TestLoggerAddCallerFail(t *testing.T) {
	errBuf := &testutils.Buffer{}
	withLogger(t, DebugLevel, opts(AddCaller(), ErrorOutput(errBuf)), func(log Logger, logs *zapcore.ObservedLogs) {
		// TODO: Use AddCallerSkip
		logImpl := log.(*logger)
		logImpl.callerSkip = 1e3

		log.Info("Failure.")
		assert.Regexp(
			t,
			`addCaller error: failed to get caller`,
			errBuf.String(),
			"Didn't find expected failure message.",
		)
		assert.Equal(
			t,
			logs.AllUntimed()[0].Entry.Message,
			"Failure.",
			"Expected original message to survive failures in runtime.Caller.")
	})
}

func TestLoggerAddStacks(t *testing.T) {
	assertHasStack := func(t testing.TB, obs zapcore.ObservedLog) {
		assert.Contains(t, obs.Entry.Stack, "zap.TestLoggerAddStacks", "Expected to find test function in stacktrace.")
	}

	withLogger(t, DebugLevel, opts(AddStacks(InfoLevel)), func(logger Logger, logs *zapcore.ObservedLogs) {
		logger.Debug("")
		assert.Empty(
			t,
			logs.AllUntimed()[0].Entry.Stack,
			"Unexpected stacktrack at DebugLevel.",
		)

		logger.Info("")
		assertHasStack(t, logs.AllUntimed()[1])

		logger.Warn("")
		assertHasStack(t, logs.AllUntimed()[2])
	})
}

func TestLoggerConcurrent(t *testing.T) {
	withLogger(t, DebugLevel, nil, func(logger Logger, logs *zapcore.ObservedLogs) {
		child := logger.With(String("foo", "bar"))

		wg := &sync.WaitGroup{}
		runConcurrently(5, 10, wg, func() {
			logger.Info("", String("foo", "bar"))
		})
		runConcurrently(5, 10, wg, func() {
			child.Info("")
		})

		wg.Wait()

		// Make sure the output doesn't contain interspersed entries.
		assert.Equal(t, 100, logs.Len(), "Unexpected number of logs written out.")
		for _, obs := range logs.AllUntimed() {
			assert.Equal(
				t,
				zapcore.ObservedLog{
					Entry:   zapcore.Entry{Level: InfoLevel},
					Context: []zapcore.Field{String("foo", "bar")},
				},
				obs,
				"Unexpected log output.",
			)
		}
	})
}
