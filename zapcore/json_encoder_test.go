// Copyright (c) 2018 Uber Technologies, Inc.
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

package zapcore_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestJSONEncodeEntry is an more "integrated" test that makes it easier to get
// coverage on the json encoder (e.g. putJSONEncoder, let alone EncodeEntry
// itself) than the tests in json_encoder_impl_test.go; it needs to be in the
// zapcore_test package, so that it can import the toplevel zap package for
// field constructor convenience.
func TestJSONEncodeEntry(t *testing.T) {
	type bar struct {
		Key string  `json:"key"`
		Val float64 `json:"val"`
	}

	type foo struct {
		A string  `json:"aee"`
		B int     `json:"bee"`
		C float64 `json:"cee"`
		D []bar   `json:"dee"`
	}

	tests := []struct {
		desc     string
		expected string
		ent      zapcore.Entry
		fields   []zapcore.Field
	}{
		{
			desc: "info entry with some fields",
			expected: `{
				"L": "info",
				"T": "2018-06-19T16:33:42.000Z",
				"N": "bob",
				"M": "lob law",
				"so": "passes",
				"answer": 42,
				"common_pie": 3.14,
				"null_value": null,
				"array_with_null_elements": [{}, null, null, 2],
				"such": {
					"aee": "lol",
					"bee": 123,
					"cee": 0.9999,
					"dee": [
						{"key": "pi", "val": 3.141592653589793},
						{"key": "tau", "val": 6.283185307179586}
					]
				}
			}`,
			ent: zapcore.Entry{
				Level:      zapcore.InfoLevel,
				Time:       time.Date(2018, 6, 19, 16, 33, 42, 99, time.UTC),
				LoggerName: "bob",
				Message:    "lob law",
			},
			fields: []zapcore.Field{
				zap.String("so", "passes"),
				zap.Int("answer", 42),
				zap.Float64("common_pie", 3.14),
				// Cover special-cased handling of nil in AddReflect() and
				// AppendReflect(). Note that for the latter, we explicitly test
				// correct results for both the nil static interface{} value
				// (`nil`), as well as the non-nil interface value with a
				// dynamic type and nil value (`(*struct{})(nil)`).
				zap.Reflect("null_value", nil),
				zap.Reflect("array_with_null_elements", []interface{}{&struct{}{}, nil, (*struct{})(nil), 2}),
				zap.Reflect("such", foo{
					A: "lol",
					B: 123,
					C: 0.9999,
					D: []bar{
						{"pi", 3.141592653589793},
						{"tau", 6.283185307179586},
					},
				}),
			},
		},
	}

	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		MessageKey:     "M",
		LevelKey:       "L",
		TimeKey:        "T",
		NameKey:        "N",
		CallerKey:      "C",
		FunctionKey:    "F",
		StacktraceKey:  "S",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			buf, err := enc.EncodeEntry(tt.ent, tt.fields)
			if assert.NoError(t, err, "Unexpected JSON encoding error.") {
				assert.JSONEq(t, tt.expected, buf.String(), "Incorrect encoded JSON entry.")
			}
			buf.Free()
		})
	}
}

func TestJSONEmptyConfig(t *testing.T) {
	tests := []struct {
		name     string
		field    zapcore.Field
		expected string
	}{
		{
			name:     "time",
			field:    zap.Time("foo", time.Unix(1591287718, 0)), // 2020-06-04 09:21:58 -0700 PDT
			expected: `{"foo": 1591287718000000000}`,
		},
		{
			name:     "duration",
			field:    zap.Duration("bar", time.Microsecond),
			expected: `{"bar": 1000}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{})

			buf, err := enc.EncodeEntry(zapcore.Entry{
				Level:      zapcore.DebugLevel,
				Time:       time.Now(),
				LoggerName: "mylogger",
				Message:    "things happened",
			}, []zapcore.Field{tt.field})
			if assert.NoError(t, err, "Unexpected JSON encoding error.") {
				assert.JSONEq(t, tt.expected, buf.String(), "Incorrect encoded JSON entry.")
			}

			buf.Free()
		})
	}
}

func TestJSONEncodeNumbersAsStrings(t *testing.T) {
	type bar struct {
		Key string  `json:"key"`
		Val float64 `json:"val"`
	}

	type foo struct {
		A string  `json:"aee"`
		B int     `json:"bee"`
		C float64 `json:"cee"`
		D []bar   `json:"dee"`
	}

	tests := []struct {
		desc     string
		expected string
		ent      zapcore.Entry
		fields   []zapcore.Field
	}{
		{
			desc: "info entry with some fields",
			expected: `{
				"L": "info",
				"T": "2018-06-19T16:33:42.000Z",
				"N": "bob",
				"M": "lob law",
				"so": "passes",
				"answer": "42",
				"common_pie": "3.14",
				"uint8": "255",
				"array_with_null_elements": [{}, null, null, 2],
				"such": {
					"aee": "lol",
					"bee": 123,
					"cee": 0.9999,
					"dee": [
						{"key": "pi", "val": 3.141592653589793},
						{"key": "tau", "val": 6.283185307179586}
					]
				}
			}`,
			ent: zapcore.Entry{
				Level:      zapcore.InfoLevel,
				Time:       time.Date(2018, 6, 19, 16, 33, 42, 99, time.UTC),
				LoggerName: "bob",
				Message:    "lob law",
			},
			fields: []zapcore.Field{
				zap.String("so", "passes"),
				zap.Int("answer", 42),
				zap.Float64("common_pie", 3.14),
				zap.Uint8("uint8", uint8(255)),
				zap.Reflect("array_with_null_elements", []interface{}{&struct{}{}, nil, (*struct{})(nil), 2}),
				zap.Reflect("such", foo{
					A: "lol",
					B: 123,
					C: 0.9999,
					D: []bar{
						{"pi", 3.141592653589793},
						{"tau", 6.283185307179586},
					},
				}),
			},
		},
	}

	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		MessageKey:       "M",
		LevelKey:         "L",
		TimeKey:          "T",
		NameKey:          "N",
		CallerKey:        "C",
		FunctionKey:      "F",
		StacktraceKey:    "S",
		EncodeLevel:      zapcore.LowercaseLevelEncoder,
		EncodeTime:       zapcore.ISO8601TimeEncoder,
		EncodeDuration:   zapcore.SecondsDurationEncoder,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		NumbersAsStrings: true,
	})

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			buf, err := enc.EncodeEntry(tt.ent, tt.fields)
			if assert.NoError(t, err, "Unexpected JSON encoding error.") {
				assert.JSONEq(t, tt.expected, buf.String(), "Incorrect encoded JSON entry.")
			}
			buf.Free()
		})
	}
}
