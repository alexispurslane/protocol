// Copyright 2025 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"encoding/json"
	"io"
)

// Marshaler is the interface implemented by types that
// can marshal themselves into valid JSON.
type Marshaler interface {
	MarshalJSON() ([]byte, error)
}

// Unmarshaler is the interface implemented by types
// that can unmarshal a JSON description of themselves.
// The input can be assumed to be a valid encoding of
// a JSON value. UnmarshalJSON must copy the JSON data
// if it wishes to retain the data after returning.
type Unmarshaler interface {
	UnmarshalJSON([]byte) error
}

// MarshalFunc function type of marshal JSON data.
//
// Default is used [json.Marshal].
type MarshalFunc func(v any) ([]byte, error)

var marshal MarshalFunc = json.Marshal

// RegiserMarshaler registers [MarshalFunc] to global marshaler.
func RegiserMarshaler(fn MarshalFunc) {
	marshal = fn
}

// UnmarshalFunc function type of unmarshal JSON data.
//
// Default is used [json.Unmarshal].
type UnmarshalFunc func(data []byte, v any) error

var unmarshal UnmarshalFunc = json.Unmarshal

// RegiserUnmarshaler registers [UnmarshalFunc] to global unmarshaler.
func RegiserUnmarshaler(fn UnmarshalFunc) {
	unmarshal = fn
}

// JSONEncoder encodes and writes to the underlying data stream.
type JSONEncoder interface {
	Encode(any) error
}

// EncoderFunc function type of [JSONEncoder].
//
// Default is used [json.NewEncoder] with SetEscapeHTML to false.
type EncoderFunc func(io.Writer) JSONEncoder

var newEncoder EncoderFunc = defaultEncoder

func defaultEncoder(w io.Writer) JSONEncoder {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc
}

// RegiserEncoder registers [EncoderFunc] to global encoder.
func RegiserEncoder(fn EncoderFunc) {
	newEncoder = fn
}

// JSONDecoder decodes and reads to the underlying data stream.
type JSONDecoder interface {
	Decode(v any) error
}

// DecoderFunc function type of [JSONDecoder].
//
// Default is used [json.NewDecoder].
type DecoderFunc func(io.Reader) JSONDecoder

var newDecoder DecoderFunc = defaultDecoder

func defaultDecoder(r io.Reader) JSONDecoder {
	dec := json.NewDecoder(r)
	return dec
}

// RegiserDecoder registers [DecoderFunc] to global decoder.
func RegiserDecoder(fn DecoderFunc) {
	newDecoder = fn
}
