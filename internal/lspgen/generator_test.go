// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBuildRegistryDuplicate(t *testing.T) {
	t.Parallel()

	model := metaModel{
		Structures: []structure{
			{Name: "foo"},
			{Name: "Foo"},
		},
	}
	_, err := buildRegistry(model)
	if err == nil {
		t.Fatalf("expected duplicate structure error")
	}
}

func TestIsSkippedType(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name string
		want bool
	}{
		"success: LSPAny": {name: "LSPAny", want: true},
		"success: LSPArray": {name: "LSPArray", want: true},
		"success: LSPObject": {name: "LSPObject", want: true},
		"success: LSPAnyOrNull": {name: "LSPAnyOrNull", want: true},
		"success: other": {name: "Other", want: false},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, isSkippedType(tt.name)); diff != "" {
				t.Fatalf("isSkippedType mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildRegistryDiscriminator(t *testing.T) {
	t.Parallel()

	model := metaModel{
		Structures: []structure{
			{
				Name: "Thing",
				Properties: []property{
					{Name: "kind", Type: metaType{Kind: metaTypeStringLiteral, StringLiteral: "thing"}},
				},
			},
		},
	}
	reg, err := buildRegistry(model)
	if err != nil {
		t.Fatalf("buildRegistry error: %v", err)
	}
	discs := reg.discriminators["Thing"]
	want := []discriminator{{JSONName: "kind", Value: "thing"}}
	if diff := cmp.Diff(want, discs); diff != "" {
		t.Errorf("discriminators mismatch (-want +got):\n%s", diff)
	}
}

func TestNewGeneratorMaps(t *testing.T) {
	t.Parallel()

	model := metaModel{}
	entries := []structMapEntry{
		{Name: "Foo", File: "foo.go"},
		{Name: "Bar", File: "bar.go"},
	}
	gen, err := newGenerator(model, entries, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	if _, ok := gen.structMapSet["Foo"]; !ok {
		t.Fatalf("structMapSet missing Foo")
	}
	if diff := cmp.Diff("bar.go", gen.structMapByName["Bar"]); diff != "" {
		t.Errorf("structMapByName mismatch (-want +got):\n%s", diff)
	}
	if gen.structMapOrder["Foo"] != 0 || gen.structMapOrder["Bar"] != 1 {
		t.Fatalf("structMapOrder mismatch: %#v", gen.structMapOrder)
	}
}

func TestCollectTypeDefsIncludesResolvedAndSpecial(t *testing.T) {
	t.Parallel()

	model := metaModel{
		Structures:  []structure{{Name: "Foo"}},
		TypeAliases: []typeAlias{{Name: "Alias", Type: metaType{Kind: metaTypeBase, Name: "string"}}},
	}
	entries := []structMapEntry{
		{Name: "Foo", File: "foo.go"},
		{Name: "ResolvedFoo", File: "foo.go"},
	}
	gen, err := newGenerator(model, entries, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	gen.needsTuple = true
	gen.needsEmptyObj = true
	gen.needsNull = true
	gen.needsURI = true

	defs, err := gen.collectTypeDefs()
	if err != nil {
		t.Fatalf("collectTypeDefs error: %v", err)
	}
	for _, name := range []string{"Foo", "ResolvedFoo", "Tuple", "EmptyObject", "Null", "DocumentURI", "URI"} {
		if _, ok := defs[name]; !ok {
			t.Fatalf("expected type def %s", name)
		}
	}
}

func TestValidateStrict(t *testing.T) {
	t.Parallel()

	gen := &generator{structMap: []structMapEntry{{Name: "Foo"}}}
	defs := map[string]typeDef{}
	if err := gen.validateStrict(defs); err == nil {
		t.Fatalf("expected strict validation error")
	}
}

func TestAliasOverrideType(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in   string
		want string
	}{
		"success: LSPAny":      {in: "LSPAny", want: "any"},
		"success: LSPArray":    {in: "LSPArray", want: "[]any"},
		"success: LSPObject":   {in: "LSPObject", want: "map[string]any"},
		"success: DocumentURI": {in: "DocumentURI", want: "uri.URI"},
		"success: URI":         {in: "URI", want: "uri.URI"},
		"success: none":        {in: "Other", want: ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, aliasOverrideType(tt.in)); diff != "" {
				t.Errorf("aliasOverrideType mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectTypeDefsSkipsLSPAliases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		aliases []typeAlias
		skip    []string
	}{
		"success: skip LSP aliases": {
			aliases: []typeAlias{
				{Name: "LSPAny", Type: metaType{Kind: metaTypeBase, Name: "string"}},
				{Name: "LSPArray", Type: metaType{Kind: metaTypeBase, Name: "string"}},
				{Name: "LSPObject", Type: metaType{Kind: metaTypeBase, Name: "string"}},
				{Name: "Other", Type: metaType{Kind: metaTypeBase, Name: "string"}},
			},
			skip: []string{"LSPAny", "LSPArray", "LSPObject"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gen, err := newGenerator(metaModel{TypeAliases: tt.aliases}, nil, config{})
			if err != nil {
				t.Fatalf("newGenerator error: %v", err)
			}

			defs, err := gen.collectTypeDefs()
			if err != nil {
				t.Fatalf("collectTypeDefs error: %v", err)
			}
			for _, skipName := range tt.skip {
				if _, ok := defs[skipName]; ok {
					t.Fatalf("did not expect type def %s", skipName)
				}
			}
			if _, ok := defs["Other"]; !ok {
				t.Fatalf("expected type def Other")
			}
		})
	}
}

func TestValidateStrictSkipsLSPAnyOrNull(t *testing.T) {
	t.Parallel()

	gen := &generator{
		structMap: []structMapEntry{
			{Name: "LSPAnyOrNull", File: "lsp.go"},
			{Name: "Other", File: "other.go"},
		},
	}
	defs := map[string]typeDef{
		"Other": {Name: "Other", Kind: typeDefStruct, Struct: &structure{Name: "Other"}},
	}
	if err := gen.validateStrict(defs); err != nil {
		t.Fatalf("validateStrict error: %v", err)
	}
}
