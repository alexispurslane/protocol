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
	"os"
	"path/filepath"
	"strings"
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

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := goPublicIdentifier(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("goPublicIdentifier(%q) mismatch (-want +got):\n%s", tt.input, diff)
			}
		})
	}
}

func TestStructFilesFromMap(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content string
		want    map[string]string
		wantErr bool
	}{
		"success: basic map": {
			content: `[
  {"name":"Foo","file":"foo.go"},
  {"name":"Bar","file":"bar.go"}
]`,
			want: map[string]string{
				"Foo": "foo.go",
				"Bar": "bar.go",
			},
		},
		"error: duplicate struct name": {
			content: `[
  {"name":"Foo","file":"foo.go"},
  {"name":"Foo","file":"bar.go"}
]`,
			wantErr: true,
		},
		"error: missing name": {
			content: `[
  {"name":"","file":"foo.go"}
]`,
			wantErr: true,
		},
		"error: empty map": {
			content: `[]`,
			wantErr: true,
		},
		"error: invalid json": {
			content: `{not-json`,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "struct-map.json")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write struct map: %v", err)
			}

			got, err := structFilesFromMap(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("structFilesFromMap() error = %v, wantErr %t", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("structFilesFromMap() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateFileSources(t *testing.T) {
	t.Parallel()

	const src = `package protocol

import (
	"fmt"
	"strings"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

type Foo struct {
	Name string
}

var _ json.MarshalerTo = (*Foo)(nil)

func (f *Foo) String() string {
	return fmt.Sprintf("%s", f.Name)
}

func (f *Foo) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return json.UnmarshalDecode(dec, &f.Name)
}

type Bar struct {
	Value string
}

func (b Bar) Upper() string {
	return strings.ToUpper(b.Value)
}

type Baz string

func helper() {}
`

	tests := map[string]struct {
		structFiles     map[string]string
		wantErr         bool
		wantContains    map[string][]string
		wantNotContains map[string][]string
	}{
		"success: split structs": {
			structFiles: map[string]string{
				"Foo": "foo.go",
				"Bar": "bar.go",
			},
			wantContains: map[string][]string{
				"foo.go": {
					"type Foo struct",
					"func (f *Foo) String() string",
					"var _ json.MarshalerTo = (*Foo)(nil)",
					`"fmt"`,
					`"github.com/go-json-experiment/json"`,
					`"github.com/go-json-experiment/json/jsontext"`,
				},
				"bar.go": {
					"type Bar struct",
					"func (b Bar) Upper() string",
					`"strings"`,
				},
				"lsp.go": {
					"type Baz string",
					"func helper()",
				},
			},
			wantNotContains: map[string][]string{
				"foo.go": {
					`"strings"`,
					"type Bar struct",
				},
				"bar.go": {
					`"fmt"`,
					`"github.com/go-json-experiment/json"`,
					"type Foo struct",
				},
				"lsp.go": {
					"type Foo struct",
					"type Bar struct",
				},
			},
		},
		"error: missing mapping": {
			structFiles: map[string]string{
				"Foo": "foo.go",
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := generateFileSources([]byte(src), tt.structFiles, "lsp.go")
			if (err != nil) != tt.wantErr {
				t.Fatalf("generateFileSources() error = %v, wantErr %t", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			for fileName, needles := range tt.wantContains {
				content, ok := got[fileName]
				if !ok {
					t.Fatalf("generateFileSources() missing file %q", fileName)
				}
				text := string(content)
				for _, needle := range needles {
					if !strings.Contains(text, needle) {
						t.Fatalf("file %q missing %q", fileName, needle)
					}
				}
			}

			for fileName, needles := range tt.wantNotContains {
				content, ok := got[fileName]
				if !ok {
					t.Fatalf("generateFileSources() missing file %q", fileName)
				}
				text := string(content)
				for _, needle := range needles {
					if strings.Contains(text, needle) {
						t.Fatalf("file %q unexpectedly contained %q", fileName, needle)
					}
				}
			}
		})
	}
}
