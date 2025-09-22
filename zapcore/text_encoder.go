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
	"encoding/base64"
	"math"
	"time"
	"unicode/utf8"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/internal/bufferpool"
	"go.uber.org/zap/internal/pool"
)

// Separator for keys
const keyComponentSep = '.'

// For JSON-escaping; see textEncoder.safeAddString below.
var _textPool = pool.New(func() *textEncoder {
	return &textEncoder{}
})

func putTextEncoder(enc *textEncoder) {
	if enc.reflectBuf != nil {
		enc.reflectBuf.Free()
	}
	enc.EncoderConfig = nil
	enc.buf = nil
	enc.spaced = false
	enc.reflectBuf = nil
	enc.reflectEnc = nil
	enc.prefixBuf.Free()
	enc.prefixBuf = nil
	enc.keyValSep = 0
	enc.elementSep = 0
	_textPool.Put(enc)
}

type textEncoder struct {
	*EncoderConfig
	buf       *buffer.Buffer
	spaced    bool // include spaces after colons and commas
	prefixBuf *buffer.Buffer
	// for encoding generic values by reflection
	reflectBuf *buffer.Buffer
	reflectEnc ReflectedEncoder
	keyValSep  byte
	elementSep byte
}

// NewTextEncoder creates a fast, low-allocation Text encoder. The encoder
// appropriately escapes all field keys and values.
//
// Note that the encoder doesn't deduplicate keys, so it's possible to produce
// a message like
//
//	"foo"="bar", "foo"="baz"
//
// This is permitted by the JSON specification, but not encouraged. Many
// libraries will ignore duplicate key-value pairs (typically keeping the last
// pair) when unmarshaling, but users should attempt to avoid adding duplicate
// keys.
func NewTextEncoder(cfg EncoderConfig) Encoder {
	return newTextEncoder(cfg, false)
}

func newTextEncoder(cfg EncoderConfig, spaced bool) *textEncoder {
	if cfg.SkipLineEnding {
		cfg.LineEnding = ""
	} else if cfg.LineEnding == "" {
		cfg.LineEnding = DefaultLineEnding
	}

	// If no EncoderConfig.NewReflectedEncoder is provided by the user, then use default
	if cfg.NewReflectedEncoder == nil {
		cfg.NewReflectedEncoder = defaultReflectedEncoder
	}

	return &textEncoder{
		EncoderConfig: &cfg,
		buf:           bufferpool.Get(),
		spaced:        spaced,
		prefixBuf:     bufferpool.Get(),
		keyValSep:     '=',
		elementSep:    ' ',
	}
}

func (enc *textEncoder) AddArray(key string, arr ArrayMarshaler) error {
	enc.addKey(key)
	return enc.AppendArray(arr)
}

func (enc *textEncoder) AddObject(key string, obj ObjectMarshaler) error {
	oldLen := enc.prefixBuf.Len()
	enc.OpenNamespace(key)
	err := enc.AppendObject(obj)
	enc.prefixBuf.Truncate(oldLen)
	return err
}

func (enc *textEncoder) AddBinary(key string, val []byte) {
	enc.AddString(key, base64.StdEncoding.EncodeToString(val))
}

func (enc *textEncoder) AddByteString(key string, val []byte) {
	enc.addKey(key)
	enc.AppendByteString(val)
}

func (enc *textEncoder) AddBool(key string, val bool) {
	enc.addKey(key)
	enc.AppendBool(val)
}

func (enc *textEncoder) AddComplex128(key string, val complex128) {
	enc.addKey(key)
	enc.AppendComplex128(val)
}

func (enc *textEncoder) AddComplex64(key string, val complex64) {
	enc.addKey(key)
	enc.AppendComplex64(val)
}

func (enc *textEncoder) AddDuration(key string, val time.Duration) {
	enc.addKey(key)
	enc.AppendDuration(val)
}

func (enc *textEncoder) AddFloat64(key string, val float64) {
	enc.addKey(key)
	enc.AppendFloat64(val)
}

func (enc *textEncoder) AddFloat32(key string, val float32) {
	enc.addKey(key)
	enc.AppendFloat32(val)
}

func (enc *textEncoder) AddInt64(key string, val int64) {
	enc.addKey(key)
	enc.AppendInt64(val)
}

func (enc *textEncoder) resetReflectBuf() {
	if enc.reflectBuf == nil {
		enc.reflectBuf = bufferpool.Get()
		enc.reflectEnc = enc.NewReflectedEncoder(enc.reflectBuf)
	} else {
		enc.reflectBuf.Reset()
	}
}

// Only invoke the standard JSON encoder if there is actually something to
// encode; otherwise write JSON null literal directly.
func (enc *textEncoder) encodeReflected(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nullLiteralBytes, nil
	}
	enc.resetReflectBuf()
	if err := enc.reflectEnc.Encode(obj); err != nil {
		return nil, err
	}
	enc.reflectBuf.TrimNewline()
	return enc.reflectBuf.Bytes(), nil
}

func (enc *textEncoder) AddReflected(key string, obj interface{}) error {
	valueBytes, err := enc.encodeReflected(obj)
	if err != nil {
		return err
	}
	enc.addKey(key)
	_, err = enc.buf.Write(valueBytes)
	return err
}

func (enc *textEncoder) OpenNamespace(key string) {
	enc.prefixBuf.AppendString(key)
	enc.prefixBuf.AppendByte(keyComponentSep)
}

func (enc *textEncoder) AddString(key, val string) {
	enc.addKey(key)
	enc.AppendString(val)
}

func (enc *textEncoder) AddTime(key string, val time.Time) {
	enc.addKey(key)
	enc.AppendTime(val)
}

func (enc *textEncoder) AddUint64(key string, val uint64) {
	enc.addKey(key)
	enc.AppendUint64(val)
}

func (enc *textEncoder) AppendArray(arr ArrayMarshaler) error {
	// Array is JSON-compliant
	enc.addElementSeparator()
	old := enc.prefixBuf
	oldKeyValSep := enc.keyValSep
	oldElementSep := enc.elementSep
	// Get new buffer to use for array elements
	enc.prefixBuf = bufferpool.Get()
	enc.keyValSep = ':'
	enc.elementSep = ','

	enc.buf.AppendString("[")
	err := arr.MarshalLogArray(enc)
	enc.buf.AppendString("]")
	enc.prefixBuf.Free()
	enc.prefixBuf = old

	enc.keyValSep = oldKeyValSep
	enc.elementSep = oldElementSep
	return err
}

func (enc *textEncoder) AppendObject(obj ObjectMarshaler) error {
	enc.addElementSeparator()
	oldLen := enc.prefixBuf.Len()
	if oldLen == 0 {
		// If no prefix, encode in JSON format,
		// Eg scenario - through AddArray/AppendArray Path
		enc.buf.AppendString("{")
	}
	err := obj.MarshalLogObject(enc)

	if oldLen == 0 {
		enc.buf.AppendString("}")
	}
	last := enc.buf.Len() - 1
	if last >= 0 {
		// Remove trailing comma or space (if enc.spaced) if necessary
		// to keep Format valid.
		// Eg - if no fields were added to the object
		if enc.spaced && enc.buf.Bytes()[last] == ' ' {
			last--
		}
		if enc.buf.Bytes()[last] == enc.elementSep {
			last--
		}
		enc.buf.Truncate(last + 1)
	}
	enc.prefixBuf.Truncate(oldLen)
	return err
}

func (enc *textEncoder) AppendBool(val bool) {
	enc.addElementSeparator()
	enc.buf.AppendBool(val)
}

func (enc *textEncoder) AppendByteString(val []byte) {
	enc.addElementSeparator()
	enc.buf.AppendByte('"')
	enc.safeAddByteString(val)
	enc.buf.AppendByte('"')
}

// appendComplex appends the encoded form of the provided complex128 value.
// precision specifies the encoding precision for the real and imaginary
// components of the complex number.
func (enc *textEncoder) appendComplex(val complex128, precision int) {
	enc.addElementSeparator()
	// Cast to a platform-independent, fixed-size type.
	r, i := float64(real(val)), float64(imag(val))
	enc.buf.AppendByte('"')
	// Because we're always in a quoted string, we can use strconv without
	// special-casing NaN and +/-Inf.
	enc.buf.AppendFloat(r, precision)
	// If imaginary part is less than 0, minus (-) sign is added by default
	// by AppendFloat.
	if i >= 0 {
		enc.buf.AppendByte('+')
	}
	enc.buf.AppendFloat(i, precision)
	enc.buf.AppendByte('i')
	enc.buf.AppendByte('"')
}

func (enc *textEncoder) AppendDuration(val time.Duration) {
	cur := enc.buf.Len()
	if e := enc.EncodeDuration; e != nil {
		e(val, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeDuration is a no-op. Fall back to nanoseconds to keep
		// JSON valid.
		enc.AppendInt64(int64(val))
	}
}

func (enc *textEncoder) AppendInt64(val int64) {
	enc.addElementSeparator()
	enc.buf.AppendInt(val)
}

func (enc *textEncoder) AppendReflected(val interface{}) error {
	valueBytes, err := enc.encodeReflected(val)
	if err != nil {
		return err
	}
	enc.addElementSeparator()
	_, err = enc.buf.Write(valueBytes)
	return err
}

func (enc *textEncoder) AppendString(val string) {
	enc.addElementSeparator()
	enc.buf.AppendByte('"')
	enc.safeAddString(val)
	enc.buf.AppendByte('"')
}

func (enc *textEncoder) AppendTimeLayout(time time.Time, layout string) {
	enc.addElementSeparator()
	enc.buf.AppendByte('"')
	enc.buf.AppendTime(time, layout)
	enc.buf.AppendByte('"')
}

func (enc *textEncoder) AppendTime(val time.Time) {
	cur := enc.buf.Len()
	if e := enc.EncodeTime; e != nil {
		e(val, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeTime is a no-op. Fall back to nanos since epoch to keep
		// output JSON valid.
		enc.AppendInt64(val.UnixNano())
	}
}

func (enc *textEncoder) AppendUint64(val uint64) {
	enc.addElementSeparator()
	enc.buf.AppendUint(val)
}

func (enc *textEncoder) AddInt(k string, v int)         { enc.AddInt64(k, int64(v)) }
func (enc *textEncoder) AddInt32(k string, v int32)     { enc.AddInt64(k, int64(v)) }
func (enc *textEncoder) AddInt16(k string, v int16)     { enc.AddInt64(k, int64(v)) }
func (enc *textEncoder) AddInt8(k string, v int8)       { enc.AddInt64(k, int64(v)) }
func (enc *textEncoder) AddUint(k string, v uint)       { enc.AddUint64(k, uint64(v)) }
func (enc *textEncoder) AddUint32(k string, v uint32)   { enc.AddUint64(k, uint64(v)) }
func (enc *textEncoder) AddUint16(k string, v uint16)   { enc.AddUint64(k, uint64(v)) }
func (enc *textEncoder) AddUint8(k string, v uint8)     { enc.AddUint64(k, uint64(v)) }
func (enc *textEncoder) AddUintptr(k string, v uintptr) { enc.AddUint64(k, uint64(v)) }
func (enc *textEncoder) AppendComplex64(v complex64)    { enc.appendComplex(complex128(v), 32) }
func (enc *textEncoder) AppendComplex128(v complex128)  { enc.appendComplex(complex128(v), 64) }
func (enc *textEncoder) AppendFloat64(v float64)        { enc.appendFloat(v, 64) }
func (enc *textEncoder) AppendFloat32(v float32)        { enc.appendFloat(float64(v), 32) }
func (enc *textEncoder) AppendInt(v int)                { enc.AppendInt64(int64(v)) }
func (enc *textEncoder) AppendInt32(v int32)            { enc.AppendInt64(int64(v)) }
func (enc *textEncoder) AppendInt16(v int16)            { enc.AppendInt64(int64(v)) }
func (enc *textEncoder) AppendInt8(v int8)              { enc.AppendInt64(int64(v)) }
func (enc *textEncoder) AppendUint(v uint)              { enc.AppendUint64(uint64(v)) }
func (enc *textEncoder) AppendUint32(v uint32)          { enc.AppendUint64(uint64(v)) }
func (enc *textEncoder) AppendUint16(v uint16)          { enc.AppendUint64(uint64(v)) }
func (enc *textEncoder) AppendUint8(v uint8)            { enc.AppendUint64(uint64(v)) }
func (enc *textEncoder) AppendUintptr(v uintptr)        { enc.AppendUint64(uint64(v)) }

func (enc *textEncoder) Clone() Encoder {
	clone := enc.clone()
	clone.buf.Write(enc.buf.Bytes())
	return clone
}

func (enc *textEncoder) clone() *textEncoder {
	clone := _textPool.Get()
	clone.EncoderConfig = enc.EncoderConfig
	clone.spaced = enc.spaced
	clone.buf = bufferpool.Get()
	clone.prefixBuf = bufferpool.Get()
	clone.keyValSep = enc.keyValSep
	clone.elementSep = enc.elementSep
	return clone
}

func (enc *textEncoder) EncodeEntry(ent Entry, fields []Field) (*buffer.Buffer, error) {
	final := enc.clone()
	// final.buf.AppendByte('{')

	if final.LevelKey != "" && final.EncodeLevel != nil {
		final.addKey(final.LevelKey)
		cur := final.buf.Len()
		final.EncodeLevel(ent.Level, final)
		if cur == final.buf.Len() {
			// User-supplied EncodeLevel was a no-op. Fall back to strings to keep
			// output JSON valid.
			final.AppendString(ent.Level.String())
		}
	}
	if final.TimeKey != "" && !ent.Time.IsZero() {
		final.AddTime(final.TimeKey, ent.Time)
	}
	if ent.LoggerName != "" && final.NameKey != "" {
		final.addKey(final.NameKey)
		cur := final.buf.Len()
		nameEncoder := final.EncodeName

		// if no name encoder provided, fall back to FullNameEncoder for backwards
		// compatibility
		if nameEncoder == nil {
			nameEncoder = FullNameEncoder
		}

		nameEncoder(ent.LoggerName, final)
		if cur == final.buf.Len() {
			// User-supplied EncodeName was a no-op. Fall back to strings to
			// keep output JSON valid.
			final.AppendString(ent.LoggerName)
		}
	}
	if ent.Caller.Defined {
		if final.CallerKey != "" {
			final.addKey(final.CallerKey)
			cur := final.buf.Len()
			final.EncodeCaller(ent.Caller, final)
			if cur == final.buf.Len() {
				// User-supplied EncodeCaller was a no-op. Fall back to strings to
				// keep output JSON valid.
				final.AppendString(ent.Caller.String())
			}
		}
		if final.FunctionKey != "" {
			final.addKey(final.FunctionKey)
			final.AppendString(ent.Caller.Function)
		}
	}
	if final.MessageKey != "" {
		final.addKey(enc.MessageKey)
		final.AppendString(ent.Message)
	}
	if enc.buf.Len() > 0 {
		final.addElementSeparator()
		final.buf.Write(enc.buf.Bytes())
	}
	addFields(final, fields)
	final.closeOpenNamespaces()
	if ent.Stack != "" && final.StacktraceKey != "" {
		final.AddString(final.StacktraceKey, ent.Stack)
	}
	// final.buf.AppendByte('}')
	final.buf.AppendString(final.LineEnding)

	ret := final.buf
	putTextEncoder(final)
	return ret, nil
}

func (enc *textEncoder) truncate() {
	enc.buf.Reset()
	enc.prefixBuf.Reset()
	enc.keyValSep = '='
	enc.elementSep = ' '
}

func (enc *textEncoder) closeOpenNamespaces() {
}

func (enc *textEncoder) addKey(key string) {
	oldLen := enc.prefixBuf.Len()
	enc.addElementSeparator()
	enc.prefixBuf.AppendString(key)
	enc.buf.AppendByte('"')
	enc.safeAddByteString(enc.prefixBuf.Bytes())
	enc.buf.AppendByte('"')
	enc.buf.AppendByte(enc.keyValSep)
	if enc.spaced {
		enc.buf.AppendByte(' ')
	}
	enc.prefixBuf.Truncate(oldLen)
}

func (enc *textEncoder) addElementSeparator() {
	last := enc.buf.Len() - 1
	if last < 0 {
		return
	}
	switch enc.buf.Bytes()[last] {
	case '{', '[', '=', ',', ' ', ':':
		return
	default:
		enc.buf.AppendByte(enc.elementSep)
		if enc.spaced {
			enc.buf.AppendByte(' ')
		}
	}
}

func (enc *textEncoder) appendFloat(val float64, bitSize int) {
	enc.addElementSeparator()
	switch {
	case math.IsNaN(val):
		enc.buf.AppendString(`"NaN"`)
	case math.IsInf(val, 1):
		enc.buf.AppendString(`"+Inf"`)
	case math.IsInf(val, -1):
		enc.buf.AppendString(`"-Inf"`)
	default:
		enc.buf.AppendFloat(val, bitSize)
	}
}

// safeAddString JSON-escapes a string and appends it to the internal buffer.
// Unlike the standard library's encoder, it doesn't attempt to protect the
// user from browser vulnerabilities or JSONP-related problems.
func (enc *textEncoder) safeAddString(s string) {
	safeAppendStringLike(
		(*buffer.Buffer).AppendString,
		utf8.DecodeRuneInString,
		enc.buf,
		s,
	)
}

// safeAddByteString is no-alloc equivalent of safeAddString(string(s)) for s []byte.
func (enc *textEncoder) safeAddByteString(s []byte) {
	safeAppendStringLike(
		(*buffer.Buffer).AppendBytes,
		utf8.DecodeRune,
		enc.buf,
		s,
	)
}
