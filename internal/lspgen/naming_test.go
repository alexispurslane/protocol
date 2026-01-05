// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGoName(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in   string
		want string
	}{
		"success: camel case": {
			in:   "textDocument",
			want: "TextDocument",
		},
		"success: initialism": {
			in:   "uri",
			want: "URI",
		},
		"success: already title": {
			in:   "DocumentURI",
			want: "DocumentURI",
		},
		"success: empty": {
			in:   "",
			want: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, goName(tt.in)); diff != "" {
				t.Errorf("goName() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSplitWords(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in   string
		want []string
	}{
		"success: camel case": {
			in:   "TextDocument",
			want: []string{"Text", "Document"},
		},
		"success: initialisms": {
			in:   "JSONRPC",
			want: []string{"JSONRPC"},
		},
		"success: empty": {
			in:   "",
			want: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, splitWords(tt.in)); diff != "" {
				t.Errorf("splitWords() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsUpper(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in   byte
		want bool
	}{
		"success: upper": {in: 'A', want: true},
		"success: lower": {in: 'z', want: false},
		"success: digit": {in: '9', want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, isUpper(tt.in)); diff != "" {
				t.Errorf("isUpper() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in   string
		want string
	}{
		"success: consonant y": {in: "story", want: "stories"},
		"success: vowel y":     {in: "day", want: "days"},
		"success: s suffix":    {in: "class", want: "classes"},
		"success: x suffix":    {in: "box", want: "boxes"},
		"success: sh suffix":   {in: "brush", want: "brushes"},
		"success: default":     {in: "car", want: "cars"},
		"success: empty":       {in: "", want: ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, pluralize(tt.in)); diff != "" {
				t.Errorf("pluralize() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestJSONTag(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name      string
		optional  bool
		omitEmpty bool
		want      string
	}{
		"success: required": {
			name:      "foo",
			optional:  false,
			omitEmpty: false,
			want:      "json:\"foo\"",
		},
		"success: optional omitzero": {
			name:      "bar",
			optional:  true,
			omitEmpty: false,
			want:      "json:\"bar,omitzero\"",
		},
		"success: optional omitempty": {
			name:      "baz",
			optional:  true,
			omitEmpty: true,
			want:      "json:\"baz,omitempty\"",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, jsonTag(tt.name, tt.optional, tt.omitEmpty)); diff != "" {
				t.Errorf("jsonTag() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStringLiteralTypeName(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in   string
		want string
	}{
		"success: path": {
			in:   "textDocument/hover",
			want: "StringLiteralTextDocumentHover",
		},
		"success: simple": {
			in:   "full",
			want: "StringLiteralFull",
		},
		"success: empty": {
			in:   "",
			want: "StringLiteralValue",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, stringLiteralTypeName(tt.in)); diff != "" {
				t.Errorf("stringLiteralTypeName() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCommonPrefixWords(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		names []string
		want  []string
	}{
		"success: shared prefix": {
			names: []string{"NotebookDocumentFilterWithNotebook", "NotebookDocumentFilterWithCells"},
			want:  []string{"Notebook", "Document", "Filter", "With"},
		},
		"success: no prefix": {
			names: []string{"FullDocument", "UnchangedDocument"},
			want:  nil,
		},
		"success: empty": {
			names: nil,
			want:  nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, commonPrefixWords(tt.names)); diff != "" {
				t.Errorf("commonPrefixWords() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTrimPrefixWords(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name   string
		prefix []string
		want   string
	}{
		"success: trims prefix": {
			name:   "NotebookDocumentFilterWithCells",
			prefix: []string{"Notebook", "Document", "Filter", "With"},
			want:   "Cells",
		},
		"success: mismatch": {
			name:   "OtherDocument",
			prefix: []string{"Notebook"},
			want:   "OtherDocument",
		},
		"success: empty": {
			name:   "",
			prefix: nil,
			want:   "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, trimPrefixWords(tt.name, tt.prefix)); diff != "" {
				t.Errorf("trimPrefixWords() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
