// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package binary implements simple translation between numbers and byte
// sequences and encoding and decoding of varints.
//
// Numbers are translated by reading and writing fixed-size values.
// A fixed-size value is either a fixed-size arithmetic
// type (bool, int8, uint8, int16, float32, complex64, ...)
// or an array or struct containing only fixed-size values.
//
// The varint functions encode and decode single integer values using
// a variable-length encoding; smaller values require fewer bytes.
// For a specification, see
// https://developers.google.com/protocol-buffers/docs/encoding.
//
// This package favors simplicity over efficiency. Clients that require
// high-performance serialization, especially for large data structures,
// should look at more advanced solutions such as the encoding/gob
// package or protocol buffers.
package binary

import (
	"errors"
	"io"
	"math"
)

// A ByteOrder specifies how to convert byte sequences into
// 16-, 32-, or 64-bit unsigned integers.
type ByteOrder interface {
	Uint16([]byte) uint16
	Uint32([]byte) uint32
	Uint64([]byte) uint64
	PutUint16([]byte, uint16)
	PutUint32([]byte, uint32)
	PutUint64([]byte, uint64)
	String() string
}

// LittleEndian is the little-endian implementation of ByteOrder.
var LittleEndian littleEndian

// BigEndian is the big-endian implementation of ByteOrder.
var BigEndian bigEndian

type littleEndian struct{}

func (littleEndian) Uint16(b []byte) uint16 {
	_ = b[1] // bounds check hint to compiler; see golang.org/issue/14808
	return uint16(b[0]) | uint16(b[1])<<8
}

func (littleEndian) PutUint16(b []byte, v uint16) {
	_ = b[1] // early bounds check to guarantee safety of writes below
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}

func (littleEndian) Uint32(b []byte) uint32 {
	_ = b[3] // bounds check hint to compiler; see golang.org/issue/14808
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func (littleEndian) PutUint32(b []byte, v uint32) {
	_ = b[3] // early bounds check to guarantee safety of writes below
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

func (littleEndian) Uint64(b []byte) uint64 {
	_ = b[7] // bounds check hint to compiler; see golang.org/issue/14808
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

func (littleEndian) PutUint64(b []byte, v uint64) {
	_ = b[7] // early bounds check to guarantee safety of writes below
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
	b[4] = byte(v >> 32)
	b[5] = byte(v >> 40)
	b[6] = byte(v >> 48)
	b[7] = byte(v >> 56)
}

func (littleEndian) String() string { return "LittleEndian" }

func (littleEndian) GoString() string { return "binary.LittleEndian" }

type bigEndian struct{}

func (bigEndian) Uint16(b []byte) uint16 {
	_ = b[1] // bounds check hint to compiler; see golang.org/issue/14808
	return uint16(b[1]) | uint16(b[0])<<8
}

func (bigEndian) PutUint16(b []byte, v uint16) {
	_ = b[1] // early bounds check to guarantee safety of writes below
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

func (bigEndian) Uint32(b []byte) uint32 {
	_ = b[3] // bounds check hint to compiler; see golang.org/issue/14808
	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
}

func (bigEndian) PutUint32(b []byte, v uint32) {
	_ = b[3] // early bounds check to guarantee safety of writes below
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}

func (bigEndian) Uint64(b []byte) uint64 {
	_ = b[7] // bounds check hint to compiler; see golang.org/issue/14808
	return uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 |
		uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56
}

func (bigEndian) PutUint64(b []byte, v uint64) {
	_ = b[7] // early bounds check to guarantee safety of writes below
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
}

func (bigEndian) String() string { return "BigEndian" }

func (bigEndian) GoString() string { return "binary.BigEndian" }

// Read reads structured binary data from r into data.
// Data must be a pointer to a fixed-size value or a slice
// of fixed-size values.
// Bytes read from r are decoded using the specified byte order
// and written to successive fields of the data.
// When decoding boolean values, a zero byte is decoded as false, and
// any other non-zero byte is decoded as true.
// When reading into structs, the field data for fields with
// blank (_) field names is skipped; i.e., blank field names
// may be used for padding.
// When reading into a struct, all non-blank fields must be exported
// or Read may panic.
//
// The error is EOF only if no bytes were read.
// If an EOF happens after reading some but not all the bytes,
// Read returns ErrUnexpectedEOF.
func Read(r io.Reader, order ByteOrder, data any) error {
	// Fast path for basic types and slices.
	if n := intDataSize(data); n != 0 {
		bs := make([]byte, n)
		if _, err := io.ReadFull(r, bs); err != nil {
			return err
		}
		switch data := data.(type) {
		case *bool:
			*data = bs[0] != 0
		case *int8:
			*data = int8(bs[0])
		case *uint8:
			*data = bs[0]
		case *int16:
			*data = int16(order.Uint16(bs))
		case *uint16:
			*data = order.Uint16(bs)
		case *int32:
			*data = int32(order.Uint32(bs))
		case *uint32:
			*data = order.Uint32(bs)
		case *int64:
			*data = int64(order.Uint64(bs))
		case *uint64:
			*data = order.Uint64(bs)
		case *float32:
			*data = math.Float32frombits(order.Uint32(bs))
		case *float64:
			*data = math.Float64frombits(order.Uint64(bs))
		case []bool:
			for i, x := range bs { // Easier to loop over the input for 8-bit values.
				data[i] = x != 0
			}
		case []int8:
			for i, x := range bs {
				data[i] = int8(x)
			}
		case []uint8:
			copy(data, bs)
		case []int16:
			for i := range data {
				data[i] = int16(order.Uint16(bs[2*i:]))
			}
		case []uint16:
			for i := range data {
				data[i] = order.Uint16(bs[2*i:])
			}
		case []int32:
			for i := range data {
				data[i] = int32(order.Uint32(bs[4*i:]))
			}
		case []uint32:
			for i := range data {
				data[i] = order.Uint32(bs[4*i:])
			}
		case []int64:
			for i := range data {
				data[i] = int64(order.Uint64(bs[8*i:]))
			}
		case []uint64:
			for i := range data {
				data[i] = order.Uint64(bs[8*i:])
			}
		case []float32:
			for i := range data {
				data[i] = math.Float32frombits(order.Uint32(bs[4*i:]))
			}
		case []float64:
			for i := range data {
				data[i] = math.Float64frombits(order.Uint64(bs[8*i:]))
			}
		default:
			n = 0 // fast path doesn't apply
		}
		if n != 0 {
			return nil
		}
	}

	// Reflect-based decoding is not supported on this port.
	return errors.New("binary.Read: invalid type")
}

// Write writes the binary representation of data into w.
// Data must be a fixed-size value or a slice of fixed-size
// values, or a pointer to such data.
// Boolean values encode as one byte: 1 for true, and 0 for false.
// Bytes written to w are encoded using the specified byte order
// and read from successive fields of the data.
// When writing structs, zero values are written for fields
// with blank (_) field names.
func Write(w io.Writer, order ByteOrder, data any) error {
	// Fast path for basic types and slices.
	if n := intDataSize(data); n != 0 {
		bs := make([]byte, n)
		switch v := data.(type) {
		case *bool:
			if *v {
				bs[0] = 1
			} else {
				bs[0] = 0
			}
		case bool:
			if v {
				bs[0] = 1
			} else {
				bs[0] = 0
			}
		case []bool:
			for i, x := range v {
				if x {
					bs[i] = 1
				} else {
					bs[i] = 0
				}
			}
		case *int8:
			bs[0] = byte(*v)
		case int8:
			bs[0] = byte(v)
		case []int8:
			for i, x := range v {
				bs[i] = byte(x)
			}
		case *uint8:
			bs[0] = *v
		case uint8:
			bs[0] = v
		case []uint8:
			bs = v
		case *int16:
			order.PutUint16(bs, uint16(*v))
		case int16:
			order.PutUint16(bs, uint16(v))
		case []int16:
			for i, x := range v {
				order.PutUint16(bs[2*i:], uint16(x))
			}
		case *uint16:
			order.PutUint16(bs, *v)
		case uint16:
			order.PutUint16(bs, v)
		case []uint16:
			for i, x := range v {
				order.PutUint16(bs[2*i:], x)
			}
		case *int32:
			order.PutUint32(bs, uint32(*v))
		case int32:
			order.PutUint32(bs, uint32(v))
		case []int32:
			for i, x := range v {
				order.PutUint32(bs[4*i:], uint32(x))
			}
		case *uint32:
			order.PutUint32(bs, *v)
		case uint32:
			order.PutUint32(bs, v)
		case []uint32:
			for i, x := range v {
				order.PutUint32(bs[4*i:], x)
			}
		case *int64:
			order.PutUint64(bs, uint64(*v))
		case int64:
			order.PutUint64(bs, uint64(v))
		case []int64:
			for i, x := range v {
				order.PutUint64(bs[8*i:], uint64(x))
			}
		case *uint64:
			order.PutUint64(bs, *v)
		case uint64:
			order.PutUint64(bs, v)
		case []uint64:
			for i, x := range v {
				order.PutUint64(bs[8*i:], x)
			}
		case *float32:
			order.PutUint32(bs, math.Float32bits(*v))
		case float32:
			order.PutUint32(bs, math.Float32bits(v))
		case []float32:
			for i, x := range v {
				order.PutUint32(bs[4*i:], math.Float32bits(x))
			}
		case *float64:
			order.PutUint64(bs, math.Float64bits(*v))
		case float64:
			order.PutUint64(bs, math.Float64bits(v))
		case []float64:
			for i, x := range v {
				order.PutUint64(bs[8*i:], math.Float64bits(x))
			}
		}
		_, err := w.Write(bs)
		return err
	}

	// Reflect-based encoding is not supported on this port.
	return errors.New("binary.Write: invalid type")
}

// Size returns how many bytes Write would generate to encode the value v, which
// must be a fixed-size value or a slice of fixed-size values, or a pointer to such data.
// If v is neither of these, Size returns -1.
func Size(v any) int {
	if n := intDataSize(v); n != 0 {
		return n
	}
	return -1
}

// intDataSize returns the size of the data required to represent the data when encoded.
// It returns zero if the type cannot be implemented by the fast path in Read or Write.
func intDataSize(data any) int {
	switch data := data.(type) {
	case bool, int8, uint8, *bool, *int8, *uint8:
		return 1
	case []bool:
		return len(data)
	case []int8:
		return len(data)
	case []uint8:
		return len(data)
	case int16, uint16, *int16, *uint16:
		return 2
	case []int16:
		return 2 * len(data)
	case []uint16:
		return 2 * len(data)
	case int32, uint32, *int32, *uint32:
		return 4
	case []int32:
		return 4 * len(data)
	case []uint32:
		return 4 * len(data)
	case int64, uint64, *int64, *uint64:
		return 8
	case []int64:
		return 8 * len(data)
	case []uint64:
		return 8 * len(data)
	case float32, *float32:
		return 4
	case float64, *float64:
		return 8
	case []float32:
		return 4 * len(data)
	case []float64:
		return 8 * len(data)
	}
	return 0
}
