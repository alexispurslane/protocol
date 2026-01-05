// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRegisterStringLiteral(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}

	if err := gen.registerStringLiteral("foo-bar"); err != nil {
		t.Fatalf("registerStringLiteral error: %v", err)
	}
	if err := gen.registerStringLiteral("foo bar"); err == nil {
		t.Fatalf("expected conflict error")
	}
}

func TestRegisterLiteralStruct(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	props := []property{{Name: "flag", Type: metaType{Kind: metaTypeBase, Name: "boolean"}}}
	name, err := gen.registerLiteralStruct(props)
	if err != nil {
		t.Fatalf("registerLiteralStruct error: %v", err)
	}
	if _, ok := gen.syntheticStructs[name]; !ok {
		t.Fatalf("expected synthetic struct %s", name)
	}

	emptyName, err := gen.registerLiteralStruct(nil)
	if err != nil {
		t.Fatalf("registerLiteralStruct empty error: %v", err)
	}
	if diff := cmp.Diff("EmptyObject", emptyName); diff != "" {
		t.Errorf("empty literal name mismatch (-want +got):\n%s", diff)
	}
}

func TestUnionCandidates(t *testing.T) {
	t.Parallel()

	model := metaModel{
		TypeAliases: []typeAlias{
			{
				Name: "AliasOrString",
				Type: metaType{Kind: metaTypeOr, Items: []metaType{
					{Kind: metaTypeBase, Name: "string"},
					{Kind: metaTypeBase, Name: "integer"},
				}},
			},
		},
	}
	gen, err := newGenerator(model, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	union := metaType{Kind: metaTypeOr, Items: []metaType{
		{Kind: metaTypeReference, Name: "AliasOrString"},
		{Kind: metaTypeBase, Name: "boolean"},
	}}

	candidates := gen.unionCandidates(union)
	if len(candidates) == 0 {
		t.Fatalf("expected union candidates")
	}
}

func TestUnionNameAndItems(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	union := metaType{Kind: metaTypeOr, Items: []metaType{{Kind: metaTypeBase, Name: "string"}, {Kind: metaTypeBase, Name: "integer"}}}
	name, items, err := gen.unionNameAndItems(union)
	if err != nil {
		t.Fatalf("unionNameAndItems error: %v", err)
	}
	if diff := cmp.Diff("StringOrInteger", name); diff != "" {
		t.Errorf("unionNameAndItems name mismatch (-want +got):\n%s", diff)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestUnionLabelHelpers(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}

	item1 := metaType{Kind: metaTypeReference, Name: "NotebookDocumentFilterWithNotebook"}
	item2 := metaType{Kind: metaTypeReference, Name: "NotebookDocumentFilterWithCells"}
	label := gen.unionLabelFromItems([]metaType{item1, item2})
	want := "NotebookDocumentFilterWithNotebookOrCells"
	if diff := cmp.Diff(want, label); diff != "" {
		t.Errorf("unionLabelFromItems mismatch (-want +got):\n%s", diff)
	}

	labels := gen.unionItemLabels([]metaType{item1, item2})
	if diff := cmp.Diff([]string{"NotebookDocumentFilterWithNotebook", "NotebookDocumentFilterWithCells"}, labels); diff != "" {
		t.Errorf("unionItemLabels mismatch (-want +got):\n%s", diff)
	}
}

func TestTypeLabelBaseAndArray(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}

	tests := map[string]struct {
		input metaType
		want  string
	}{
		"success: base string": {input: metaType{Kind: metaTypeBase, Name: "string"}, want: "String"},
		"success: base null":   {input: metaType{Kind: metaTypeBase, Name: "null"}, want: "Null"},
		"success: array":       {input: metaType{Kind: metaTypeArray, Element: &metaType{Kind: metaTypeReference, Name: "TextDocumentEdit"}}, want: "TextDocumentEdits"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, gen.typeLabel(tt.input)); diff != "" {
				t.Errorf("typeLabel mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBaseTypeLabel(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  string
	}{
		"success: string":       {input: "string", want: "String"},
		"success: boolean":      {input: "boolean", want: "Boolean"},
		"success: document uri": {input: "DocumentUri", want: "DocumentURI"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, baseTypeLabel(tt.input)); diff != "" {
				t.Errorf("baseTypeLabel mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLiteralNamingHelpers(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}

	props := []property{{Name: "flag", Type: metaType{Kind: metaTypeBase, Name: "boolean"}}}
	signature := gen.literalSignature(props)
	name := literalStructNameFromSignature(signature)
	if name == "" {
		t.Fatalf("expected literal struct name")
	}
	if fnv32a(signature) == 0 {
		t.Fatalf("expected non-zero hash")
	}
}

func TestRegisterUnionTypeWarning(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}

	itemsA := []metaType{{Kind: metaTypeBase, Name: "string"}, {Kind: metaTypeBase, Name: "integer"}}
	itemsB := []metaType{{Kind: metaTypeBase, Name: "string"}}
	gen.registerUnionType("StringOrInteger", itemsA)
	gen.registerUnionType("StringOrInteger", itemsB)
	if len(gen.warnings) == 0 {
		t.Fatalf("expected warning for conflicting union items")
	}
}

func TestMaybeRegisterPropertyAlias(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}

	prop := property{Name: "data", Type: metaType{Kind: metaTypeReference, Name: "LSPAny"}}
	gen.maybeRegisterPropertyAlias("Foo", prop)
	if _, ok := gen.syntheticAliases["FooData"]; !ok {
		t.Fatalf("expected FooData alias")
	}
}

func TestCollectSyntheticTypesFlags(t *testing.T) {
	t.Parallel()

	model := metaModel{
		Structures: []structure{
			{
				Name: "Thing",
				Properties: []property{
					{Name: "range", Type: metaType{Kind: metaTypeOr, Items: []metaType{{Kind: metaTypeBase, Name: "boolean"}, {Kind: metaTypeLiteral, Literal: &literalType{Properties: nil}}}}},
				},
			},
		},
		TypeAliases: []typeAlias{{Name: "TupleAlias", Type: metaType{Kind: metaTypeTuple}}},
	}

	gen, err := newGenerator(model, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	if !gen.needsEmptyObj {
		t.Fatalf("expected needsEmptyObj")
	}
	if !gen.needsTuple {
		t.Fatalf("expected needsTuple")
	}
	if len(gen.unionDefs) == 0 {
		t.Fatalf("expected unionDefs")
	}
}

func TestCollectSyntheticTypesSkipsLSPAnyOrNullUnion(t *testing.T) {
	t.Parallel()

	model := metaModel{
		Requests: []request{
			{
				TypeName: "ExecuteCommandRequest",
				Result: &metaType{
					Kind: metaTypeOr,
					Items: []metaType{
						{Kind: metaTypeReference, Name: "LSPAny"},
						{Kind: metaTypeBase, Name: "null"},
					},
				},
			},
		},
		TypeAliases: []typeAlias{
			{Name: "LSPAny", Type: metaType{Kind: metaTypeOr, Items: []metaType{{Kind: metaTypeBase, Name: "string"}, {Kind: metaTypeBase, Name: "null"}}}},
		},
	}

	gen, err := newGenerator(model, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	if _, ok := gen.unionDefs["LSPAnyOrNull"]; ok {
		t.Fatalf("did not expect LSPAnyOrNull union")
	}
}

func TestVisitTypeWithOptions(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}

	orType := metaType{
		Kind: metaTypeOr,
		Items: []metaType{{Kind: metaTypeStringLiteral, StringLiteral: "foo"}},
	}

	if err := gen.visitTypeWithOptions(orType, false); err != nil {
		t.Fatalf("visitTypeWithOptions error: %v", err)
	}
	if len(gen.unionDefs) != 0 {
		t.Fatalf("expected no unionDefs, got %d", len(gen.unionDefs))
	}
	if _, ok := gen.stringLiterals[stringLiteralTypeName("foo")]; !ok {
		t.Fatalf("expected string literal registration")
	}

	gen2, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	if err := gen2.visitTypeWithOptions(orType, true); err != nil {
		t.Fatalf("visitTypeWithOptions error: %v", err)
	}
	if len(gen2.unionDefs) == 0 {
		t.Fatalf("expected unionDefs")
	}
}

func TestRegisterRegistrationOptions(t *testing.T) {
	t.Parallel()

	model := metaModel{
		Requests: []request{
			{
				TypeName:            "ColorPresentationRequest",
				RegistrationOptions: &metaType{Kind: metaTypeAnd, Items: []metaType{{Kind: metaTypeReference, Name: "WorkDoneProgressOptions"}, {Kind: metaTypeReference, Name: "TextDocumentRegistrationOptions"}}},
			},
		},
	}
	gen, err := newGenerator(model, []structMapEntry{{Name: "ColorPresentationRegistrationOptions", File: "capabilities_server.go"}}, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	if _, ok := gen.syntheticStructs["ColorPresentationRegistrationOptions"]; !ok {
		t.Fatalf("expected synthetic registration options")
	}
}

func TestSlicesEqual(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		left  []string
		right []string
		want  bool
	}{
		"success: equal":     {left: []string{"a", "b"}, right: []string{"a", "b"}, want: true},
		"success: not equal": {left: []string{"a"}, right: []string{"b"}, want: false},
		"success: length":    {left: []string{"a"}, right: []string{"a", "b"}, want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, slicesEqual(tt.left, tt.right)); diff != "" {
				t.Errorf("slicesEqual mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
