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

package benchmarks

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/testutils"
	"go.uber.org/zap/zapcore"
)

func fakeSugarFields() zap.Ctx {
	return zap.Ctx{
		"error":          errExample,
		"int":            1,
		"int64":          2,
		"float":          3.0,
		"string":         "four!",
		"stringer":       zap.DebugLevel,
		"bool":           true,
		"time":           time.Unix(0, 0),
		"duration":       time.Second,
		"another string": "done!",
	}
}

func newSugarLogger(lvl zapcore.Level, options ...zap.Option) *zap.SugaredLogger {
	return zap.Sugar(zap.New(zapcore.WriterFacility(
		benchEncoder(),
		&testutils.Discarder{},
		lvl,
	), options...))
}

func BenchmarkZapSugarDisabledLevelsWithoutFields(b *testing.B) {
	logger := newSugarLogger(zap.ErrorLevel)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Should be discarded.")
		}
	})
}

func BenchmarkZapSugarDisabledLevelsAccumulatedContext(b *testing.B) {
	logger := newSugarLogger(zap.ErrorLevel, zap.Fields(fakeFields()...))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Should be discarded.")
		}
	})
}

func BenchmarkZapSugarDisabledLevelsAddingFields(b *testing.B) {
	logger := newSugarLogger(zap.ErrorLevel)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.InfoWith("Should be discarded.", fakeSugarFields())
		}
	})
}

func BenchmarkZapSugarAddingFields(b *testing.B) {
	logger := newSugarLogger(zap.DebugLevel)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.InfoWith("Go fast.", fakeSugarFields())
		}
	})
}

func BenchmarkZapSugarWithAccumulatedContext(b *testing.B) {
	logger := newSugarLogger(zap.DebugLevel, zap.Fields(fakeFields()...))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Go really fast.")
		}
	})
}

func BenchmarkZapSugarWithoutFields(b *testing.B) {
	logger := newSugarLogger(zap.DebugLevel)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Go fast.")
		}
	})
}
