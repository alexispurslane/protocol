// Copyright 2025 The Go Language Server Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGoPublicIdentifier(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  string
	}{
		"initialism: DocumentUri -> DocumentURI": {
			input: "DocumentUri",
			want:  "DocumentURI",
		},
		"initialism: uri -> URI": {
			input: "uri",
			want:  "URI",
		},
		"initialism: rootUri -> RootURI": {
			input: "rootUri",
			want:  "RootURI",
		},
		"initialism: resultId -> ResultID": {
			input: "resultId",
			want:  "ResultID",
		},
		"initialism: previousResultId -> PreviousResultID": {
			input: "previousResultId",
			want:  "PreviousResultID",
		},
		"plural initialism: previousResultIds -> PreviousResultIDs": {
			input: "previousResultIds",
			want:  "PreviousResultIDs",
		},
		"plural initialism: documentUris -> DocumentURIs": {
			input: "documentUris",
			want:  "DocumentURIs",
		},
		"preserve acronym: LSPAny": {
			input: "LSPAny",
			want:  "LSPAny",
		},
		"preserve acronym: ABAP": {
			input: "ABAP",
			want:  "ABAP",
		},
		"snake_case with initialism: foo_bar_id -> FooBarID": {
			input: "foo_bar_id",
			want:  "FooBarID",
		},
		"already Go-ish: TextDocumentIdentifier": {
			input: "TextDocumentIdentifier",
			want:  "TextDocumentIdentifier",
		},
		"digits in initialism: utf8 -> UTF8": {
			input: "utf8",
			want:  "UTF8",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := goPublicIdentifier(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("goPublicIdentifier(%q) mismatch (-want +got):\n%s", tc.input, diff)
			}
		})
	}
}

func TestStructNamesFromSource(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input   string
		want    []string
		wantErr bool
	}{
		"success: single struct": {
			input: "package p\n\ntype Foo struct {\n\tA int\n}\n",
			want:  []string{"Foo"},
		},
		"success: grouped and mixed types": {
			input: "package p\n\ntype (\n\tAlpha struct{}\n\tBeta int\n\tGamma struct{ X string }\n)\n\ntype Delta interface{}\n\ntype Epsilon struct{}\n",
			want:  []string{"Alpha", "Gamma", "Epsilon"},
		},
		"success: multiple declarations": {
			input: "package p\n\ntype First struct{}\n\ntype Second = string\n\ntype Third struct{}\n",
			want:  []string{"First", "Third"},
		},
		"error: invalid source": {
			input:   "package p\n\ntype",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := structNamesFromSource([]byte(tc.input))
			if (err != nil) != tc.wantErr {
				t.Fatalf("structNamesFromSource error mismatch: wantErr=%t gotErr=%t err=%v", tc.wantErr, err != nil, err)
			}
			if tc.wantErr {
				return
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("structNamesFromSource result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
