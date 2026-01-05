// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRenderStructOptional(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{packageName: "protocol"})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	str := structure{
		Name:       "Thing",
		Properties: []property{{Name: "name", Type: metaType{Kind: metaTypeBase, Name: "string"}, Optional: true}},
	}
	def := typeDef{Name: "Thing", Kind: typeDefStruct, Struct: &str}

	imports := newFileImports()
	var b strings.Builder
	if err := gen.renderType(&b, imports, def); err != nil {
		t.Fatalf("renderType error: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "json:\"name,omitempty\"") {
		t.Fatalf("expected omitempty tag, got: %s", out)
	}
	if !strings.Contains(out, "Name string") {
		t.Fatalf("expected non-pointer field, got: %s", out)
	}
}

func TestRenderStructFieldSpacing(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{packageName: "protocol"})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	str := structure{
		Name: "Thing",
		Properties: []property{
			{Name: "first", Documentation: "First field.", Type: metaType{Kind: metaTypeBase, Name: "string"}},
			{Name: "second", Documentation: "Second field.", Type: metaType{Kind: metaTypeBase, Name: "string"}},
		},
	}
	def := typeDef{Name: "Thing", Kind: typeDefStruct, Struct: &str}

	var b strings.Builder
	if err := gen.renderType(&b, newFileImports(), def); err != nil {
		t.Fatalf("renderType error: %v", err)
	}
	out := b.String()
	want := "\t// First field.\n\n\tFirst string `json:\"first\"`\n\t// Second field.\n\n\tSecond string `json:\"second\"`"
	if !strings.Contains(out, want) {
		t.Fatalf("expected blank line between field docs and field, got: %s", out)
	}
}

func TestRenderEnum(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	enum := enumeration{
		Name:   "TokenType",
		Type:   metaType{Kind: metaTypeBase, Name: "string"},
		Values: []enumerationValue{{Name: "alpha", Value: []byte(`"alpha"`)}},
	}
	def := typeDef{Name: "TokenType", Kind: typeDefEnum, Enum: &enum}

	var b strings.Builder
	if err := gen.renderEnum(&b, def); err != nil {
		t.Fatalf("renderEnum error: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "type TokenType string") {
		t.Fatalf("expected enum type, got: %s", out)
	}
	if !strings.Contains(out, "TokenTypeAlpha") {
		t.Fatalf("expected enum value const, got: %s", out)
	}
}

func TestRenderAliasOverride(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	alias := typeAlias{Name: "DocumentURI"}
	def := typeDef{Name: "DocumentURI", Kind: typeDefAlias, Alias: &alias}

	imports := newFileImports()
	var b strings.Builder
	if err := gen.renderAlias(&b, imports, def); err != nil {
		t.Fatalf("renderAlias error: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "type DocumentURI = uri.URI") {
		t.Fatalf("expected override alias, got: %s", out)
	}
	if _, ok := imports.paths["go.lsp.dev/uri"]; !ok {
		t.Fatalf("expected uri import")
	}
}

func TestRenderUnion(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	union := unionDef{Name: "StringOrInteger", Items: []metaType{{Kind: metaTypeBase, Name: "string"}, {Kind: metaTypeBase, Name: "integer"}}}
	def := typeDef{Name: "StringOrInteger", Kind: typeDefUnion, Union: &union}

	imports := newFileImports()
	var b strings.Builder
	if err := gen.renderUnion(&b, imports, def); err != nil {
		t.Fatalf("renderUnion error: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "type StringOrInteger struct") {
		t.Fatalf("expected union struct, got: %s", out)
	}
	if !strings.Contains(out, "MarshalJSON") {
		t.Fatalf("expected MarshalJSON, got: %s", out)
	}
}

func TestRenderUnionNullUsesNew(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	union := unionDef{
		Name: "LSPAnyOrNull",
		Items: []metaType{
			{Kind: metaTypeReference, Name: "LSPAny"},
			{Kind: metaTypeBase, Name: "null"},
		},
	}
	def := typeDef{Name: "LSPAnyOrNull", Kind: typeDefUnion, Union: &union}

	imports := newFileImports()
	var b strings.Builder
	if err := gen.renderUnion(&b, imports, def); err != nil {
		t.Fatalf("renderUnion error: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "new(Null)") {
		t.Fatalf("expected new(Null) in output, got: %s", out)
	}
	if strings.Contains(out, "&any{}") {
		t.Fatalf("unexpected &any{} in output: %s", out)
	}
}

func TestRenderStringLiteral(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	def := typeDef{Name: "StringLiteralFull", Kind: typeDefStringLiteral, StringValue: "full"}

	imports := newFileImports()
	var b strings.Builder
	if err := gen.renderStringLiteral(&b, imports, def); err != nil {
		t.Fatalf("renderStringLiteral error: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "const StringLiteralFullValue") {
		t.Fatalf("expected string literal const, got: %s", out)
	}
	if !strings.Contains(out, "MarshalJSON") {
		t.Fatalf("expected MarshalJSON, got: %s", out)
	}
}

func TestRenderFileHeader(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{packageName: "protocol", fallbackFile: "lsp_gen.go"})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	gen.needsNull = true
	defs := []typeDef{{Name: "Null", Kind: typeDefNull}}
	data, err := gen.renderFile("lsp_gen.go", defs)
	if err != nil {
		t.Fatalf("renderFile error: %v", err)
	}
	if !strings.Contains(string(data), "package protocol") {
		t.Fatalf("expected package line in output")
	}
}

func TestRenderUnionHelpers(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{packageName: "protocol", fallbackFile: "lsp_gen.go"})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	gen.unionDefs["StringOrInteger"] = unionDef{Name: "StringOrInteger", Items: []metaType{{Kind: metaTypeBase, Name: "string"}}}

	data, err := gen.renderFile("lsp_gen.go", nil)
	if err != nil {
		t.Fatalf("renderFile error: %v", err)
	}
	if !strings.Contains(string(data), "unionKindMatches") {
		t.Fatalf("expected union helper functions in output")
	}
}

func TestRenderTupleAndEmptyObject(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}

	tupleDef := typeDef{Name: "Tuple", Kind: typeDefTuple}
	emptyDef := typeDef{Name: "EmptyObject", Kind: typeDefEmptyObject}

	imports := newFileImports()
	var b strings.Builder
	if err := gen.renderType(&b, imports, tupleDef); err != nil {
		t.Fatalf("renderTuple error: %v", err)
	}
	if err := gen.renderType(&b, imports, emptyDef); err != nil {
		t.Fatalf("renderEmptyObject error: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "type Tuple []any") {
		t.Fatalf("expected Tuple type in output")
	}
	if !strings.Contains(out, "type EmptyObject struct{}") {
		t.Fatalf("expected EmptyObject type in output")
	}
}

func TestRenderResolvedAlias(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}

	def := typeDef{Name: "ResolvedFoo", Kind: typeDefResolvedStruct, ResolvedFrom: "Foo", Struct: &structure{Name: "Foo"}}
	var b strings.Builder
	if err := gen.renderType(&b, newFileImports(), def); err != nil {
		t.Fatalf("renderResolvedAlias error: %v", err)
	}
	if !strings.Contains(b.String(), "type ResolvedFoo = Foo") {
		t.Fatalf("expected resolved alias in output")
	}
}

func TestAdjustStructDoc(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name string
		doc  string
		want string
	}{
		"success: prefix A article": {
			name: "CodeAction",
			doc:  "A code action represents a change that can be performed in code.",
			want: "CodeAction a code action represents a change that can be performed in code.",
		},
		"success: prefix An article": {
			name: "Range",
			doc:  "An LSP range represents a text range.",
			want: "Range an LSP range represents a text range.",
		},
		"success: prefix The article": {
			name: "ClientCapabilities",
			doc:  "The client capabilities define what a client can do.",
			want: "ClientCapabilities the client capabilities define what a client can do.",
		},
		"success: already prefixed": {
			name: "DocumentSymbol",
			doc:  "DocumentSymbol represents programming constructs.",
			want: "DocumentSymbol represents programming constructs.",
		},
		"success: non article prefixed": {
			name: "CompletionOptions",
			doc:  "Options to control completion.",
			want: "CompletionOptions options to control completion.",
		},
		"success: leading tag unchanged": {
			name: "CompletionOptions",
			doc:  "@since 3.0.0",
			want: "@since 3.0.0",
		},
		"success: represents lowercase": {
			name: "Diagnostic",
			doc:  "represents a diagnostic, such as a compiler error or warning.",
			want: "Diagnostic represents a diagnostic, such as a compiler error or warning.",
		},
		"success: represents uppercase": {
			name: "Diagnostic",
			doc:  "Represents a diagnostic, such as a compiler error or warning.",
			want: "Diagnostic represents a diagnostic, such as a compiler error or warning.",
		},
		"success: preserves acronym": {
			name: "DocumentURI",
			doc:  "URI represents a document identifier.",
			want: "DocumentURI URI represents a document identifier.",
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, adjustStructDoc(tt.name, tt.doc)); diff != "" {
				t.Fatalf("adjustStructDoc mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLowerLeadingWord(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in   string
		want string
	}{
		"success: lowercase first word": {
			in:   "Represents a thing",
			want: "represents a thing",
		},
		"success: keep acronym": {
			in:   "URI represents a thing",
			want: "URI represents a thing",
		},
		"success: single letter": {
			in:   "A",
			want: "a",
		},
		"success: empty": {
			in:   "",
			want: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, lowerLeadingWord(tt.in)); diff != "" {
				t.Fatalf("lowerLeadingWord mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
