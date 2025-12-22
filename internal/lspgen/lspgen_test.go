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

func TestStructFilesByName(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		files   map[string]string
		want    map[string]string
		wantErr bool
	}{
		"success: collects structs and ignores non-protocol/test/lsp files": {
			files: map[string]string{
				"base.go":       "package protocol\n\ntype Foo struct{}\n",
				"more.go":       "package protocol\n\ntype Bar struct{}\n\ntype Baz struct{}\n",
				"lsp.go":        "package protocol\n\ntype ShouldIgnore struct{}\n",
				"extra_test.go": "package protocol\n\ntype TestOnly struct{}\n",
				"other.go":      "package other\n\ntype Other struct{}\n",
			},
			want: map[string]string{
				"Foo": "base.go",
				"Bar": "more.go",
				"Baz": "more.go",
			},
		},
		"error: duplicate struct name in multiple files": {
			files: map[string]string{
				"a.go": "package protocol\n\ntype Dup struct{}\n",
				"b.go": "package protocol\n\ntype Dup struct{}\n",
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			for filename, contents := range tc.files {
				path := filepath.Join(dir, filename)
				if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
					t.Fatalf("write test file %s: %v", filename, err)
				}
			}

			got, err := structFilesByName(dir)
			if (err != nil) != tc.wantErr {
				t.Fatalf("structFilesByName error mismatch: wantErr=%t gotErr=%t err=%v", tc.wantErr, err != nil, err)
			}
			if tc.wantErr {
				return
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("structFilesByName result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStructOutputsFromSource(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input       string
		filesByName map[string]string
		defaultFile string
		includeFile bool
		want        []structOutput
		wantErr     bool
	}{
		"success: uses mapping and default file when enabled": {
			input: "package p\n\ntype Foo struct{}\n\ntype Bar struct{}\n",
			filesByName: map[string]string{
				"Foo": "base.go",
			},
			defaultFile: "lsp.go",
			includeFile: true,
			want: []structOutput{
				{Name: "Foo", File: "base.go"},
				{Name: "Bar", File: "lsp.go"},
			},
		},
		"success: omits file when mapping disabled": {
			input: "package p\n\ntype Foo struct{}\n\ntype Bar struct{}\n",
			filesByName: map[string]string{
				"Foo": "base.go",
			},
			defaultFile: "lsp.go",
			includeFile: false,
			want: []structOutput{
				{Name: "Foo"},
				{Name: "Bar"},
			},
		},
		"error: invalid source": {
			input:       "package p\n\ntype",
			includeFile: true,
			wantErr:     true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := structOutputsFromSource([]byte(tc.input), tc.filesByName, tc.defaultFile, tc.includeFile)
			if (err != nil) != tc.wantErr {
				t.Fatalf("structOutputsFromSource error mismatch: wantErr=%t gotErr=%t err=%v", tc.wantErr, err != nil, err)
			}
			if tc.wantErr {
				return
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("structOutputsFromSource result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveOutputPath(t *testing.T) {
	t.Parallel()

	absPath := filepath.Join(t.TempDir(), "structs.json")

	tests := map[string]struct {
		repoRoot string
		input    string
		want     string
	}{
		"success: empty input returns empty output": {
			repoRoot: "/repo/root",
			input:    "",
			want:     "",
		},
		"success: relative input joins with repo root": {
			repoRoot: "/repo/root",
			input:    "structs.json",
			want:     filepath.Clean(filepath.Join("/repo/root", "structs.json")),
		},
		"success: absolute input preserves path": {
			repoRoot: "/repo/root",
			input:    absPath,
			want:     filepath.Clean(absPath),
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := resolveOutputPath(tc.repoRoot, tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("resolveOutputPath result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWriteStructOutputs(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		outputs  []structOutput
		pathFunc func(t *testing.T) string
		want     string
		wantErr  bool
	}{
		"success: writes JSON with newline": {
			outputs: []structOutput{
				{Name: "Foo", File: "base.go"},
				{Name: "Bar"},
			},
			pathFunc: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "structs.json")
			},
			want:    "[{\"name\":\"Foo\",\"file\":\"base.go\"},{\"name\":\"Bar\"}]\n",
			wantErr: false,
		},
		"error: path is a directory": {
			outputs: []structOutput{
				{Name: "Only"},
			},
			pathFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := tc.pathFunc(t)
			err := writeStructOutputs(path, tc.outputs)
			if (err != nil) != tc.wantErr {
				t.Fatalf("writeStructOutputs error mismatch: wantErr=%t gotErr=%t err=%v", tc.wantErr, err != nil, err)
			}
			if tc.wantErr {
				return
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read output file: %v", err)
			}
			if diff := cmp.Diff(tc.want, string(got)); diff != "" {
				t.Fatalf("writeStructOutputs output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStructFilesFromMap(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		contents string
		want     map[string]string
		wantErr  bool
	}{
		"success: parses name to file mapping": {
			contents: `[{"name":"Foo","file":"base.go"},{"name":"Bar","file":"general.go"}]`,
			want: map[string]string{
				"Foo": "base.go",
				"Bar": "general.go",
			},
		},
		"error: duplicate struct name with different files": {
			contents: `[{"name":"Foo","file":"base.go"},{"name":"Foo","file":"other.go"}]`,
			wantErr:  true,
		},
		"error: missing file mapping": {
			contents: `[{"name":"Foo","file":""}]`,
			wantErr:  true,
		},
		"error: missing struct name": {
			contents: `[{"name":"","file":"base.go"}]`,
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "struct-map.json")
			if err := os.WriteFile(path, []byte(tc.contents), 0o644); err != nil {
				t.Fatalf("write struct map: %v", err)
			}

			got, err := structFilesFromMap(path)
			if (err != nil) != tc.wantErr {
				t.Fatalf("structFilesFromMap error mismatch: wantErr=%t gotErr=%t err=%v", tc.wantErr, err != nil, err)
			}
			if tc.wantErr {
				return
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("structFilesFromMap result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateFileSources(t *testing.T) {
	t.Parallel()

	const source = `package protocol

import "fmt"

type Foo struct{}

func (f *Foo) String() string {
	return fmt.Sprint("foo")
}

type Bar struct{}

type Baz = string

func (b Baz) String() string {
	return fmt.Sprint("baz")
}
`

	tests := map[string]struct {
		structFiles map[string]string
		wantErr     bool
	}{
		"success: splits structs into mapped files": {
			structFiles: map[string]string{
				"Foo": "foo.go",
				"Bar": "bar.go",
			},
		},
		"error: missing struct mapping": {
			structFiles: map[string]string{},
			wantErr:     true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := generateFileSources([]byte(source), tc.structFiles, "lsp.go")
			if (err != nil) != tc.wantErr {
				t.Fatalf("generateFileSources error mismatch: wantErr=%t gotErr=%t err=%v", tc.wantErr, err != nil, err)
			}
			if tc.wantErr {
				return
			}

			foo := string(got["foo.go"])
			if !strings.Contains(foo, "type Foo struct") {
				t.Fatalf("foo.go missing Foo struct")
			}
			if !strings.Contains(foo, "func (f *Foo) String") {
				t.Fatalf("foo.go missing Foo method")
			}
			if !strings.Contains(foo, "\nimport ") || !strings.Contains(foo, "\"fmt\"") {
				t.Fatalf("foo.go missing fmt import")
			}

			bar := string(got["bar.go"])
			if !strings.Contains(bar, "type Bar struct") {
				t.Fatalf("bar.go missing Bar struct")
			}
			if strings.Contains(bar, "\nimport ") {
				t.Fatalf("bar.go should not include imports")
			}

			lsp := string(got["lsp.go"])
			if !strings.Contains(lsp, "type Baz = string") {
				t.Fatalf("lsp.go missing Baz alias")
			}
			if !strings.Contains(lsp, "func (b Baz) String") {
				t.Fatalf("lsp.go missing Baz method")
			}
			if !strings.Contains(lsp, "\nimport ") || !strings.Contains(lsp, "\"fmt\"") {
				t.Fatalf("lsp.go missing fmt import")
			}
			if strings.Contains(lsp, "type Foo struct") || strings.Contains(lsp, "type Bar struct") {
				t.Fatalf("lsp.go should not include mapped struct declarations")
			}
		})
	}
}
