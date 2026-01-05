// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/google/go-cmp/cmp"
)

func TestMetaTypeUnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    string
		want     metaType
		wantErr  bool
		errMatch string
	}{
		"success: base": {
			input: `{"kind":"base","name":"string"}`,
			want:  metaType{Kind: metaTypeBase, Name: "string"},
		},
		"success: reference": {
			input: `{"kind":"reference","name":"WorkspaceFolder"}`,
			want:  metaType{Kind: metaTypeReference, Name: "WorkspaceFolder"},
		},
		"success: array": {
			input: `{"kind":"array","element":{"kind":"base","name":"integer"}}`,
			want: metaType{
				Kind:    metaTypeArray,
				Element: &metaType{Kind: metaTypeBase, Name: "integer"},
			},
		},
		"success: map": {
			input: `{"kind":"map","key":{"kind":"base","name":"string"},"value":{"kind":"reference","name":"Location"}}`,
			want: metaType{
				Kind:  metaTypeMap,
				Key:   &metaType{Kind: metaTypeBase, Name: "string"},
				Value: &metaType{Kind: metaTypeReference, Name: "Location"},
			},
		},
		"success: or": {
			input: `{"kind":"or","items":[{"kind":"base","name":"string"},{"kind":"base","name":"integer"}]}`,
			want: metaType{
				Kind: metaTypeOr,
				Items: []metaType{
					{Kind: metaTypeBase, Name: "string"},
					{Kind: metaTypeBase, Name: "integer"},
				},
			},
		},
		"success: and": {
			input: `{"kind":"and","items":[{"kind":"reference","name":"A"},{"kind":"reference","name":"B"}]}`,
			want: metaType{
				Kind: metaTypeAnd,
				Items: []metaType{
					{Kind: metaTypeReference, Name: "A"},
					{Kind: metaTypeReference, Name: "B"},
				},
			},
		},
		"success: tuple": {
			input: `{"kind":"tuple","items":[{"kind":"base","name":"string"}]}`,
			want: metaType{
				Kind:  metaTypeTuple,
				Items: []metaType{{Kind: metaTypeBase, Name: "string"}},
			},
		},
		"success: literal": {
			input: `{"kind":"literal","value":{"properties":[{"name":"flag","type":{"kind":"base","name":"boolean"}}]}}`,
			want: metaType{
				Kind:    metaTypeLiteral,
				Literal: &literalType{Properties: []property{{Name: "flag", Type: metaType{Kind: metaTypeBase, Name: "boolean"}}}},
			},
		},
		"success: string literal": {
			input: `{"kind":"stringLiteral","value":"kind"}`,
			want:  metaType{Kind: metaTypeStringLiteral, StringLiteral: "kind"},
		},
		"error: unknown kind": {
			input:    `{"kind":"unknown"}`,
			wantErr:  true,
			errMatch: "unsupported",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got metaType
			err := json.Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal error mismatch: %v", err)
			}
			if tt.wantErr {
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("metaType mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
