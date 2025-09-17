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

package zapcore

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextEncoderObjectFields(t *testing.T) {
	tests := []struct {
		desc     string
		expected string
		f        func(Encoder)
	}{
		{"binary", `"k"="YWIxMg=="`, func(e Encoder) { e.AddBinary("k", []byte("ab12")) }},
		{"bool", `"k\\"=true`, func(e Encoder) { e.AddBool(`k\`, true) }}, // test key escaping once
		{"bool", `"k"=true`, func(e Encoder) { e.AddBool("k", true) }},
		{"bool", `"k"=false`, func(e Encoder) { e.AddBool("k", false) }},
		{"byteString", `"k"="v\\"`, func(e Encoder) { e.AddByteString(`k`, []byte(`v\`)) }},
		{"byteString", `"k"="v"`, func(e Encoder) { e.AddByteString("k", []byte("v")) }},
		{"byteString", `"k"=""`, func(e Encoder) { e.AddByteString("k", []byte{}) }},
		{"byteString", `"k"=""`, func(e Encoder) { e.AddByteString("k", nil) }},
		{"complex128", `"k"="1+2i"`, func(e Encoder) { e.AddComplex128("k", 1+2i) }},
		{"complex128/negative_i", `"k"="1-2i"`, func(e Encoder) { e.AddComplex128("k", 1-2i) }},
		{"complex64", `"k"="1+2i"`, func(e Encoder) { e.AddComplex64("k", 1+2i) }},
		{"complex64/negative_i", `"k"="1-2i"`, func(e Encoder) { e.AddComplex64("k", 1-2i) }},
		{"complex64", `"k"="2.71+3.14i"`, func(e Encoder) { e.AddComplex64("k", 2.71+3.14i) }},
		{"duration", `"k"=0.000000001`, func(e Encoder) { e.AddDuration("k", 1) }},
		{"duration/negative", `"k"=-0.000000001`, func(e Encoder) { e.AddDuration("k", -1) }},
		{"float64", `"k"=1`, func(e Encoder) { e.AddFloat64("k", 1.0) }},
		{"float64", `"k"=10000000000`, func(e Encoder) { e.AddFloat64("k", 1e10) }},
		{"float64", `"k"="NaN"`, func(e Encoder) { e.AddFloat64("k", math.NaN()) }},
		{"float64", `"k"="+Inf"`, func(e Encoder) { e.AddFloat64("k", math.Inf(1)) }},
		{"float64", `"k"="-Inf"`, func(e Encoder) { e.AddFloat64("k", math.Inf(-1)) }},
		{"float64/pi", `"k"=3.141592653589793`, func(e Encoder) { e.AddFloat64("k", math.Pi) }},
		{"float32", `"k"=1`, func(e Encoder) { e.AddFloat32("k", 1.0) }},
		{"float32", `"k"=2.71`, func(e Encoder) { e.AddFloat32("k", 2.71) }},
		{"float32", `"k"=0.1`, func(e Encoder) { e.AddFloat32("k", 0.1) }},
		{"float32", `"k"=10000000000`, func(e Encoder) { e.AddFloat32("k", 1e10) }},
		{"float32", `"k"="NaN"`, func(e Encoder) { e.AddFloat32("k", float32(math.NaN())) }},
		{"float32", `"k"="+Inf"`, func(e Encoder) { e.AddFloat32("k", float32(math.Inf(1))) }},
		{"float32", `"k"="-Inf"`, func(e Encoder) { e.AddFloat32("k", float32(math.Inf(-1))) }},
		{"float32/pi", `"k"=3.1415927`, func(e Encoder) { e.AddFloat32("k", math.Pi) }},
		{"int", `"k"=42`, func(e Encoder) { e.AddInt("k", 42) }},
		{"int64", `"k"=42`, func(e Encoder) { e.AddInt64("k", 42) }},
		{"int64/min", `"k"=-9223372036854775808`, func(e Encoder) { e.AddInt64("k", math.MinInt64) }},
		{"int64/max", `"k"=9223372036854775807`, func(e Encoder) { e.AddInt64("k", math.MaxInt64) }},
		{"int32", `"k"=42`, func(e Encoder) { e.AddInt32("k", 42) }},
		{"int32/min", `"k"=-2147483648`, func(e Encoder) { e.AddInt32("k", math.MinInt32) }},
		{"int32/max", `"k"=2147483647`, func(e Encoder) { e.AddInt32("k", math.MaxInt32) }},
		{"int16", `"k"=42`, func(e Encoder) { e.AddInt16("k", 42) }},
		{"int16/min", `"k"=-32768`, func(e Encoder) { e.AddInt16("k", math.MinInt16) }},
		{"int16/max", `"k"=32767`, func(e Encoder) { e.AddInt16("k", math.MaxInt16) }},
		{"int8", `"k"=42`, func(e Encoder) { e.AddInt8("k", 42) }},
		{"int8/min", `"k"=-128`, func(e Encoder) { e.AddInt8("k", math.MinInt8) }},
		{"int8/max", `"k"=127`, func(e Encoder) { e.AddInt8("k", math.MaxInt8) }},
		{"string", `"k"="v\\"`, func(e Encoder) { e.AddString(`k`, `v\`) }},
		{"string", `"k"="v"`, func(e Encoder) { e.AddString("k", "v") }},
		{"string", `"k"=""`, func(e Encoder) { e.AddString("k", "") }},
		{"time", `"k"=1`, func(e Encoder) { e.AddTime("k", time.Unix(1, 0)) }},
		{"uint", `"k"=42`, func(e Encoder) { e.AddUint("k", 42) }},
		{"uint64", `"k"=42`, func(e Encoder) { e.AddUint64("k", 42) }},
		{"uint64/max", `"k"=18446744073709551615`, func(e Encoder) { e.AddUint64("k", math.MaxUint64) }},
		{"uint32", `"k"=42`, func(e Encoder) { e.AddUint32("k", 42) }},
		{"uint32/max", `"k"=4294967295`, func(e Encoder) { e.AddUint32("k", math.MaxUint32) }},
		{"uint16", `"k"=42`, func(e Encoder) { e.AddUint16("k", 42) }},
		{"uint16/max", `"k"=65535`, func(e Encoder) { e.AddUint16("k", math.MaxUint16) }},
		{"uint8", `"k"=42`, func(e Encoder) { e.AddUint8("k", 42) }},
		{"uint8/max", `"k"=255`, func(e Encoder) { e.AddUint8("k", math.MaxUint8) }},
		{"uintptr", `"k"=42`, func(e Encoder) { e.AddUintptr("k", 42) }},
		{
			desc:     "object (success)",
			expected: `"k.loggable"="yes"`,
			f: func(e Encoder) {
				assert.NoError(t, e.AddObject("k", loggable{true}), "Unexpected error calling MarshalLogObject.")
			},
		},
		{
			desc:     "object (error)",
			expected: "",
			f: func(e Encoder) {
				assert.Error(t, e.AddObject("k", loggable{false}), "Expected an error calling MarshalLogObject.")
			},
		},
		{
			desc:     "object (with nested array)",
			expected: `"turducken.ducks"=[{"in":"chicken"},{"in":"chicken"}]`,
			f: func(e Encoder) {
				assert.NoError(
					t,
					e.AddObject("turducken", turducken{}),
					"Unexpected error calling MarshalLogObject with nested ObjectMarshalers and ArrayMarshalers.",
				)
			},
		},
		{
			desc:     "array (with nested object)",
			expected: `"turduckens"=[{"ducks":[{"in":"chicken"},{"in":"chicken"}]},{"ducks":[{"in":"chicken"},{"in":"chicken"}]}]`,
			f: func(e Encoder) {
				assert.NoError(
					t,
					e.AddArray("turduckens", turduckens(2)),
					"Unexpected error calling MarshalLogObject with nested ObjectMarshalers and ArrayMarshalers.",
				)
			},
		},
		{
			desc:     "array (success)",
			expected: `"k"=[true]`,
			f: func(e Encoder) {
				assert.NoError(t, e.AddArray(`k`, loggable{true}), "Unexpected error calling MarshalLogArray.")
			},
		},
		{
			desc:     "array (error)",
			expected: `"k"=[]`,
			f: func(e Encoder) {
				assert.Error(t, e.AddArray("k", loggable{false}), "Expected an error calling MarshalLogArray.")
			},
		},
		{
			desc:     "reflect (success)",
			expected: `"k"={"escape":"<&>","loggable":"yes"}`,
			f: func(e Encoder) {
				assert.NoError(t, e.AddReflected("k", map[string]string{"escape": "<&>", "loggable": "yes"}), "Unexpected error JSON-serializing a map.")
			},
		},
		{
			desc:     "reflect (failure)",
			expected: "",
			f: func(e Encoder) {
				assert.Error(t, e.AddReflected("k", noJSON{}), "Unexpected success JSON-serializing a noJSON.")
			},
		},
		{
			desc: "namespace",
			// EncodeEntry is responsible for closing all open namespaces.
			expected: `"outermost.outer.foo"=1 "outermost.outer.inner.foo"=2 "outermost.outer.inner.innermost.foo"=3`,
			f: func(e Encoder) {
				e.OpenNamespace("outermost")
				e.OpenNamespace("outer")
				e.AddInt("foo", 1)
				e.OpenNamespace("inner")
				e.AddInt("foo", 2)
				e.OpenNamespace("innermost")
				e.AddInt("foo", 3)
			},
		},
		{
			desc:     "object (no nested namespace)",
			expected: `"obj.obj-out"="obj-outside-namespace" "not-obj"="should-be-outside-obj"`,
			f: func(e Encoder) {
				assert.NoError(t, e.AddObject("obj", maybeNamespace{false}))
				e.AddString("not-obj", "should-be-outside-obj")
			},
		},
		{
			desc:     "object (with nested namespace)",
			expected: `"obj.obj-out"="obj-outside-namespace" "obj.obj-namespace.obj-in"="obj-inside-namespace" "not-obj"="should-be-outside-obj"`,
			f: func(e Encoder) {
				assert.NoError(t, e.AddObject("obj", maybeNamespace{true}))
				e.AddString("not-obj", "should-be-outside-obj")
			},
		},
		{
			desc:     "multiple open namespaces",
			expected: `"k.foo"=1 "k.middle.foo"=2 "k.middle.inner.foo"=3`,
			f: func(e Encoder) {
				err := e.AddObject("k", ObjectMarshalerFunc(func(enc ObjectEncoder) error {
					e.AddInt("foo", 1)
					e.OpenNamespace("middle")
					e.AddInt("foo", 2)
					e.OpenNamespace("inner")
					e.AddInt("foo", 3)
					return nil
				}))
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			assertTextOutput(t, _defaultEncoderConfig, tt.expected, tt.f)
		})
	}
}

func TestTextEncoderArrays(t *testing.T) {
	tests := []struct {
		desc     string
		expected string // expect f to be called twice
		f        func(ArrayEncoder)
	}{
		{"bool", `[true,true]`, func(e ArrayEncoder) { e.AppendBool(true) }},
		{"byteString", `["k","k"]`, func(e ArrayEncoder) { e.AppendByteString([]byte("k")) }},
		{"byteString", `["k\\","k\\"]`, func(e ArrayEncoder) { e.AppendByteString([]byte(`k\`)) }},
		{"complex128", `["1+2i","1+2i"]`, func(e ArrayEncoder) { e.AppendComplex128(1 + 2i) }},
		{"complex64", `["1+2i","1+2i"]`, func(e ArrayEncoder) { e.AppendComplex64(1 + 2i) }},
		{"durations", `[0.000000002,0.000000002]`, func(e ArrayEncoder) { e.AppendDuration(2) }},
		{"float64", `[3.14,3.14]`, func(e ArrayEncoder) { e.AppendFloat64(3.14) }},
		{"float32", `[3.14,3.14]`, func(e ArrayEncoder) { e.AppendFloat32(3.14) }},
		{"int", `[42,42]`, func(e ArrayEncoder) { e.AppendInt(42) }},
		{"int64", `[42,42]`, func(e ArrayEncoder) { e.AppendInt64(42) }},
		{"int32", `[42,42]`, func(e ArrayEncoder) { e.AppendInt32(42) }},
		{"int16", `[42,42]`, func(e ArrayEncoder) { e.AppendInt16(42) }},
		{"int8", `[42,42]`, func(e ArrayEncoder) { e.AppendInt8(42) }},
		{"string", `["k","k"]`, func(e ArrayEncoder) { e.AppendString("k") }},
		{"string", `["k\\","k\\"]`, func(e ArrayEncoder) { e.AppendString(`k\`) }},
		{"times", `[1,1]`, func(e ArrayEncoder) { e.AppendTime(time.Unix(1, 0)) }},
		{"uint", `[42,42]`, func(e ArrayEncoder) { e.AppendUint(42) }},
		{"uint64", `[42,42]`, func(e ArrayEncoder) { e.AppendUint64(42) }},
		{"uint32", `[42,42]`, func(e ArrayEncoder) { e.AppendUint32(42) }},
		{"uint16", `[42,42]`, func(e ArrayEncoder) { e.AppendUint16(42) }},
		{"uint8", `[42,42]`, func(e ArrayEncoder) { e.AppendUint8(42) }},
		{"uintptr", `[42,42]`, func(e ArrayEncoder) { e.AppendUintptr(42) }},
		{
			desc:     "arrays (success)",
			expected: `[[true],[true]]`,
			f: func(arr ArrayEncoder) {
				assert.NoError(t, arr.AppendArray(ArrayMarshalerFunc(func(inner ArrayEncoder) error {
					inner.AppendBool(true)
					return nil
				})), "Unexpected error appending an array.")
			},
		},
		{
			desc:     "arrays (error)",
			expected: `[[true],[true]]`,
			f: func(arr ArrayEncoder) {
				assert.Error(t, arr.AppendArray(ArrayMarshalerFunc(func(inner ArrayEncoder) error {
					inner.AppendBool(true)
					return errors.New("fail")
				})), "Expected an error appending an array.")
			},
		},
		{
			desc:     "objects (success)",
			expected: `[{"loggable":"yes"},{"loggable":"yes"}]`,
			f: func(arr ArrayEncoder) {
				assert.NoError(t, arr.AppendObject(loggable{true}), "Unexpected error appending an object.")
			},
		},
		{
			desc:     "objects (error)",
			expected: `[{},{}]`,
			f: func(arr ArrayEncoder) {
				assert.Error(t, arr.AppendObject(loggable{false}), "Expected an error appending an object.")
			},
		},
		{
			desc:     "reflect (success)",
			expected: `[{"foo":5},{"foo":5}]`,
			f: func(arr ArrayEncoder) {
				assert.NoError(
					t,
					arr.AppendReflected(map[string]int{"foo": 5}),
					"Unexpected an error appending an object with reflection.",
				)
			},
		},
		{
			desc:     "reflect (error)",
			expected: `[]`,
			f: func(arr ArrayEncoder) {
				assert.Error(
					t,
					arr.AppendReflected(noJSON{}),
					"Unexpected an error appending an object with reflection.",
				)
			},
		},
		{
			desc:     "object (no nested namespace) then string",
			expected: `[{"obj-out":"obj-outside-namespace"},"should-be-outside-obj",{"obj-out":"obj-outside-namespace"},"should-be-outside-obj"]`,
			f: func(arr ArrayEncoder) {
				assert.NoError(t, arr.AppendObject(maybeNamespace{false}))
				arr.AppendString("should-be-outside-obj")
			},
		},
		{
			desc:     "object (with nested namespace) then string",
			expected: `[{"obj-out":"obj-outside-namespace","obj-namespace.obj-in":"obj-inside-namespace"},"should-be-outside-obj",{"obj-out":"obj-outside-namespace","obj-namespace.obj-in":"obj-inside-namespace"},"should-be-outside-obj"]`,
			f: func(arr ArrayEncoder) {
				assert.NoError(t, arr.AppendObject(maybeNamespace{true}))
				arr.AppendString("should-be-outside-obj")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			f := func(enc Encoder) error {
				return enc.AddArray("array", ArrayMarshalerFunc(func(arr ArrayEncoder) error {
					tt.f(arr)
					tt.f(arr)
					return nil
				}))
			}
			assertTextOutput(t, _defaultEncoderConfig, `"array"=`+tt.expected, func(enc Encoder) {
				err := f(enc)
				assert.NoError(t, err, "Unexpected error adding array to JSON encoder.")
			})
		})
	}
}

func TestTextEncoderUserWithNestedAddr(t *testing.T) {
	u0 := user{
		name: "Sagar",
		age:  30,
		addrs: []addr{
			{city: "Pune", zip: "411001", geo: geo{lat: 18.5204, lon: 73.8567}},
			{city: "Mumbai", zip: "400001", geo: geo{lat: 19.0760, lon: 72.8777}},
		},
		curAddr: addr{
			city: "Pune", zip: "411001",
			geo: geo{lat: 18.5204, lon: 73.8567},
		},
	}

	tests := []struct {
		desc     string
		expected string
		f        func(Encoder)
	}{
		{
			desc: "two users; addr as nested object via AddObject",
			expected: `"user.name"="Sagar" "user.age"=30 ` +
				`"user.addrs"=[{"city":"Pune","zip":"411001","geo.lat":18.5204,"geo.lon":73.8567},` +
				`{"city":"Mumbai","zip":"400001","geo.lat":19.076,"geo.lon":72.8777}] ` +
				`"user.cur_addr.city"="Pune" "user.cur_addr.zip"="411001" ` +
				`"user.cur_addr.geo.lat"=18.5204 "user.cur_addr.geo.lon"=73.8567`,

			f: func(e Encoder) {
				require.NoError(t, e.AddObject("user", u0))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			assertTextOutput(t, _defaultEncoderConfig, tt.expected, tt.f)
		})
	}
}

func TestTextEncoderTimeArrays(t *testing.T) {
	times := []time.Time{
		time.Unix(1008720000, 0).UTC(), // 2001-12-19
		time.Unix(1040169600, 0).UTC(), // 2002-12-18
		time.Unix(1071619200, 0).UTC(), // 2003-12-17
	}

	tests := []struct {
		desc    string
		encoder TimeEncoder
		want    string
	}{
		{
			desc:    "epoch",
			encoder: EpochTimeEncoder,
			want:    `[1008720000,1040169600,1071619200]`,
		},
		{
			desc:    "epoch millis",
			encoder: EpochMillisTimeEncoder,
			want:    `[1008720000000,1040169600000,1071619200000]`,
		},
		{
			desc:    "iso8601",
			encoder: ISO8601TimeEncoder,
			want:    `["2001-12-19T00:00:00.000Z","2002-12-18T00:00:00.000Z","2003-12-17T00:00:00.000Z"]`,
		},
		{
			desc:    "rfc3339",
			encoder: RFC3339TimeEncoder,
			want:    `["2001-12-19T00:00:00Z","2002-12-18T00:00:00Z","2003-12-17T00:00:00Z"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			cfg := _defaultEncoderConfig
			cfg.EncodeTime = tt.encoder

			enc := NewTextEncoder(cfg).(*textEncoder)
			err := enc.AddArray("array", ArrayMarshalerFunc(func(arr ArrayEncoder) error {
				for _, time := range times {
					arr.AppendTime(time)
				}
				return nil
			}))
			assert.NoError(t, err)
			assert.Equal(t, `"array"=`+tt.want, enc.buf.String())
		})
	}
}

func assertTextOutput(t testing.TB, cfg EncoderConfig, expected string, f func(Encoder)) {
	enc := NewTextEncoder(cfg).(*textEncoder)
	f(enc)
	assert.Equal(t, expected, enc.buf.String(), "Unexpected encoder output after adding.")

	enc.truncate()
	enc.AddString("foo", "bar")
	f(enc)
	expectedPrefix := `"foo"="bar"`
	if expected != "" {
		// If we expect output, it should be space-separated from the previous
		// field.
		expectedPrefix += " "
	}
	assert.Equal(t, expectedPrefix+expected, enc.buf.String(), "Unexpected encoder output after adding as a second field.")
}

// ----- nested leaf under addr -----
type geo struct {
	lat float64
	lon float64
}

func (g geo) MarshalLogObject(enc ObjectEncoder) error {
	enc.AddFloat64("lat", g.lat)
	enc.AddFloat64("lon", g.lon)
	return nil
}

var _ ObjectMarshaler = (*geo)(nil)

// ----- addr includes nested geo -----
type addr struct {
	city string
	zip  string
	geo  geo
}

func (a addr) MarshalLogObject(enc ObjectEncoder) error {
	enc.AddString("city", a.city)
	enc.AddString("zip", a.zip)
	return enc.AddObject("geo", a.geo) // nested object inside addr
}

var _ ObjectMarshaler = (*addr)(nil)

type user struct {
	name    string
	age     int
	addrs   []addr // array of nested objects
	curAddr addr   // single nested object
}

// array marshaler for []addr
type addrList []addr

func (al addrList) MarshalLogArray(arr ArrayEncoder) error {
	for _, a := range al {
		if err := arr.AppendObject(a); err != nil {
			return err
		}
	}
	return nil
}

func (u user) MarshalLogObject(enc ObjectEncoder) error {
	enc.AddString("name", u.name)
	enc.AddInt("age", u.age)
	if err := enc.AddArray("addrs", addrList(u.addrs)); err != nil {
		return err
	}
	if err := enc.AddObject("cur_addr", u.curAddr); err != nil {
		return err
	}

	return nil
}

var _ ObjectMarshaler = (*user)(nil)
