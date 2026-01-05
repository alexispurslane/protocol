// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGoType(t *testing.T) {
	t.Parallel()

	model := metaModel{
		Structures:  []structure{{Name: "Thing"}},
		TypeAliases: []typeAlias{{Name: "Alias", Type: metaType{Kind: metaTypeBase, Name: "string"}}},
	}
	gen, err := newGenerator(model, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}

	tests := map[string]struct {
		input metaType
		want  string
	}{
		"success: base":            {input: metaType{Kind: metaTypeBase, Name: "string"}, want: "string"},
		"success: reference":       {input: metaType{Kind: metaTypeReference, Name: "Thing"}, want: "Thing"},
		"success: lsp any":         {input: metaType{Kind: metaTypeReference, Name: "LSPAny"}, want: "any"},
		"success: lsp array":       {input: metaType{Kind: metaTypeReference, Name: "LSPArray"}, want: "[]any"},
		"success: lsp object":      {input: metaType{Kind: metaTypeReference, Name: "LSPObject"}, want: "map[string]any"},
		"success: array":           {input: metaType{Kind: metaTypeArray, Element: &metaType{Kind: metaTypeReference, Name: "Thing"}}, want: "[]Thing"},
		"success: map":             {input: metaType{Kind: metaTypeMap, Value: &metaType{Kind: metaTypeReference, Name: "Thing"}}, want: "map[string]Thing"},
		"success: or":              {input: metaType{Kind: metaTypeOr, Items: []metaType{{Kind: metaTypeBase, Name: "string"}, {Kind: metaTypeBase, Name: "integer"}}}, want: "StringOrInteger"},
		"success: lsp any or null": {input: metaType{Kind: metaTypeOr, Items: []metaType{{Kind: metaTypeReference, Name: "LSPAny"}, {Kind: metaTypeBase, Name: "null"}}}, want: "any"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := gen.goType(tt.input, typeOptions{})
			if err != nil {
				t.Fatalf("goType error: %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("goType mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestShouldPointer(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}

	tests := map[string]struct {
		input metaType
		want  bool
	}{
		"success: base string": {input: metaType{Kind: metaTypeBase, Name: "string"}, want: false},
		"success: array":       {input: metaType{Kind: metaTypeArray, Element: &metaType{Kind: metaTypeBase, Name: "string"}}, want: false},
		"success: null":        {input: metaType{Kind: metaTypeBase, Name: "null"}, want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, gen.shouldPointer(tt.input)); diff != "" {
				t.Errorf("shouldPointer mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsPrimitiveGoType(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}

	tests := map[string]struct {
		input metaType
		want  bool
	}{
		"success: string":       {input: metaType{Kind: metaTypeBase, Name: "string"}, want: true},
		"success: boolean":      {input: metaType{Kind: metaTypeBase, Name: "boolean"}, want: true},
		"success: integer":      {input: metaType{Kind: metaTypeBase, Name: "integer"}, want: true},
		"success: uinteger":     {input: metaType{Kind: metaTypeBase, Name: "uinteger"}, want: true},
		"success: decimal":      {input: metaType{Kind: metaTypeBase, Name: "decimal"}, want: true},
		"success: document uri": {input: metaType{Kind: metaTypeBase, Name: "DocumentUri"}, want: false},
		"success: null":         {input: metaType{Kind: metaTypeBase, Name: "null"}, want: false},
		"success: reference":    {input: metaType{Kind: metaTypeReference, Name: "Thing"}, want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, gen.isPrimitiveGoType(tt.input)); diff != "" {
				t.Errorf("isPrimitiveGoType mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsLSPAnyOrNull(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}

	tests := map[string]struct {
		items []metaType
		want  bool
	}{
		"success: lsp any or null": {
			items: []metaType{{Kind: metaTypeReference, Name: "LSPAny"}, {Kind: metaTypeBase, Name: "null"}},
			want:  true,
		},
		"success: lsp any only": {
			items: []metaType{{Kind: metaTypeReference, Name: "LSPAny"}},
			want:  false,
		},
		"success: null only": {
			items: []metaType{{Kind: metaTypeBase, Name: "null"}},
			want:  false,
		},
		"success: mixed extras": {
			items: []metaType{{Kind: metaTypeReference, Name: "LSPAny"}, {Kind: metaTypeBase, Name: "null"}, {Kind: metaTypeBase, Name: "string"}},
			want:  false,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, gen.isLSPAnyOrNull(tt.items)); diff != "" {
				t.Errorf("isLSPAnyOrNull mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBaseGoTypeFlags(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	if _, err := gen.baseGoType("DocumentUri"); err != nil {
		t.Fatalf("baseGoType error: %v", err)
	}
	if !gen.needsURI {
		t.Fatalf("expected needsURI flag")
	}
	if _, err := gen.baseGoType("null"); err != nil {
		t.Fatalf("baseGoType error: %v", err)
	}
	if !gen.needsNull {
		t.Fatalf("expected needsNull flag")
	}
}

func TestPropertyGoTypeAliases(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	prop := property{Name: "data", Type: metaType{Kind: metaTypeReference, Name: "LSPAny"}}
	got, err := gen.propertyGoType("Foo", prop, typeOptions{})
	if err != nil {
		t.Fatalf("propertyGoType error: %v", err)
	}
	if diff := cmp.Diff("FooData", got); diff != "" {
		t.Errorf("propertyGoType mismatch (-want +got):\n%s", diff)
	}
}

func TestReferenceOverrideType(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name string
		want string
	}{
		"success: LSPAny": {name: "LSPAny", want: "any"},
		"success: LSPArray": {name: "LSPArray", want: "[]any"},
		"success: LSPObject": {name: "LSPObject", want: "map[string]any"},
		"success: other": {name: "Other", want: ""},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, referenceOverrideType(tt.name)); diff != "" {
				t.Errorf("referenceOverrideType mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestKindMask(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	if diff := cmp.Diff(kindMaskString, gen.kindMask(metaType{Kind: metaTypeBase, Name: "string"})); diff != "" {
		t.Errorf("kindMask mismatch (-want +got):\n%s", diff)
	}
}

func TestDiscriminatorsForType(t *testing.T) {
	t.Parallel()

	model := metaModel{
		Structures: []structure{
			{
				Name:       "Thing",
				Properties: []property{{Name: "kind", Type: metaType{Kind: metaTypeStringLiteral, StringLiteral: "thing"}}},
			},
		},
	}
	gen, err := newGenerator(model, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	discs := gen.discriminatorsForType(metaType{Kind: metaTypeReference, Name: "Thing"})
	want := []discriminator{{JSONName: "kind", Value: "thing"}}
	if diff := cmp.Diff(want, discs); diff != "" {
		t.Errorf("discriminatorsForType mismatch (-want +got):\n%s", diff)
	}
}

func TestDiscriminatorsForSyntheticType(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	gen.syntheticStructs["Synth"] = structure{
		Name:       "Synth",
		Properties: []property{{Name: "kind", Type: metaType{Kind: metaTypeStringLiteral, StringLiteral: "synth"}}},
	}

	discs := gen.discriminatorsForType(metaType{Kind: metaTypeReference, Name: "Synth"})
	want := []discriminator{{JSONName: "kind", Value: "synth"}}
	if diff := cmp.Diff(want, discs); diff != "" {
		t.Errorf("discriminatorsForType mismatch (-want +got):\n%s", diff)
	}
}

func TestUnionVariants(t *testing.T) {
	t.Parallel()

	gen, err := newGenerator(metaModel{}, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	union := unionDef{Name: "StringOrInteger", Items: []metaType{{Kind: metaTypeBase, Name: "string"}, {Kind: metaTypeBase, Name: "integer"}}}
	variants, err := gen.unionVariants(union)
	if err != nil {
		t.Fatalf("unionVariants error: %v", err)
	}
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}
}

func TestDocHelpers(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		doc        string
		since      string
		deprecated string
		proposed   bool
		want       []string
	}{
		"success: doc and since": {
			doc:   "Hello",
			since: "3.0.0",
			want:  []string{"Hello.", "Since 3.0.0."},
		},
		"success: multiline since and deprecated": {
			doc:        "Hello",
			since:      "3.18.0 - support for relative patterns.\nrelative patterns depends on the client capability\n`textDocuments.filters.relativePatternSupport`",
			deprecated: "Use NewThing.\nIt depends on X",
			want: []string{
				"Hello.",
				"Since 3.18.0 - support for relative patterns.",
				"relative patterns depends on the client capability.",
				"`textDocuments.filters.relativePatternSupport`.",
				"Deprecated. Use NewThing.",
				"It depends on X.",
			},
		},
		"success: since proposed": {
			since: "3.18.0 - proposed",
			want:  []string{"Since 3.18.0, Proposed."},
		},
		"success: existing since tag prevents duplication": {
			doc:   "Trim trailing whitespace on a line.\n\n@since 3.15.0",
			since: "3.15.0",
			want: []string{
				"Trim trailing whitespace on a line.",
				"",
				"Since 3.15.0.",
			},
		},
		"success: empty": {
			doc:   "",
			since: "",
			want:  nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := docLines(tt.doc, tt.since, tt.deprecated, tt.proposed)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("docLines mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNormalizeDocTags(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lines []string
		want  []string
		state docTagState
	}{
		"success: since conversion": {
			lines: []string{"@since 3.0.0", "hello"},
			want:  []string{"Since 3.0.0", "hello"},
			state: docTagState{hasSince: true},
		},
		"success: duplicate since ignored": {
			lines: []string{"@since 3.0.0", "@since 3.0.0", "hello"},
			want:  []string{"Since 3.0.0", "hello"},
			state: docTagState{hasSince: true},
		},
		"success: proposed conversion": {
			lines: []string{"@proposed"},
			want:  []string{"Proposed"},
			state: docTagState{hasProposed: true},
		},
		"success: proposed suppressed by since": {
			lines: []string{"@since 3.18.0 - proposed", "@proposed"},
			want:  []string{"Since 3.18.0, Proposed"},
			state: docTagState{hasSince: true, hasProposed: true},
		},
		"success: deprecated detection": {
			lines: []string{"@deprecated use NewThing"},
			want:  []string{"@deprecated use NewThing"},
			state: docTagState{hasDeprecated: true},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, state := normalizeDocTags(tt.lines)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("normalizeDocTags lines mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.state, state); diff != "" {
				t.Errorf("normalizeDocTags state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatSinceLine(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value    string
		want     string
		proposed bool
	}{
		"success: plain": {
			value:    "3.15.0",
			want:     "Since 3.15.0",
			proposed: false,
		},
		"success: proposed suffix": {
			value:    "3.18.0 - proposed",
			want:     "Since 3.18.0, Proposed",
			proposed: true,
		},
		"success: proposed suffix with period": {
			value:    "3.18.0 - proposed.",
			want:     "Since 3.18.0, Proposed",
			proposed: true,
		},
		"success: empty": {
			value:    "",
			want:     "Since",
			proposed: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, proposed := formatSinceLine(tt.value)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("formatSinceLine mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.proposed, proposed); diff != "" {
				t.Errorf("formatSinceLine proposed mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatSinceLines(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value    string
		want     []string
		proposed bool
	}{
		"success: multiline": {
			value:    "3.18.0 - proposed\nextra details",
			want:     []string{"Since 3.18.0, Proposed", "extra details"},
			proposed: true,
		},
		"success: empty": {
			value:    "",
			want:     nil,
			proposed: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, proposed := formatSinceLines(tt.value)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("formatSinceLines mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.proposed, proposed); diff != "" {
				t.Errorf("formatSinceLines proposed mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatDeprecatedLine(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value string
		want  string
	}{
		"success: empty": {
			value: "",
			want:  "Deprecated",
		},
		"success: plain": {
			value: "Use NewThing.",
			want:  "Deprecated. Use NewThing.",
		},
		"success: whitespace": {
			value: "  Use NewThing ",
			want:  "Deprecated. Use NewThing",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, formatDeprecatedLine(tt.value)); diff != "" {
				t.Errorf("formatDeprecatedLine mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatDeprecatedLines(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value string
		want  []string
	}{
		"success: empty": {
			value: "",
			want:  nil,
		},
		"success: single line": {
			value: "Use NewThing.",
			want:  []string{"Deprecated. Use NewThing."},
		},
		"success: multiline": {
			value: "Use NewThing.\nIt depends on X",
			want: []string{
				"Deprecated. Use NewThing.",
				"It depends on X",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, formatDeprecatedLines(tt.value)); diff != "" {
				t.Errorf("formatDeprecatedLines mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCommentLines(t *testing.T) {
	t.Parallel()

	lines := []string{"Hello.", "", "World."}
	want := []string{"// Hello.", "//", "// World."}
	if diff := cmp.Diff(want, commentLines(lines)); diff != "" {
		t.Errorf("commentLines mismatch (-want +got):\n%s", diff)
	}
}

func TestFormatConstName(t *testing.T) {
	t.Parallel()

	if diff := cmp.Diff("SymbolKindFile", formatConstName("SymbolKind", "file")); diff != "" {
		t.Errorf("formatConstName mismatch (-want +got):\n%s", diff)
	}
}

func TestJoinNonEmpty(t *testing.T) {
	t.Parallel()

	if diff := cmp.Diff("a\n\n", joinNonEmpty([]string{"a", ""})); diff != "" {
		t.Errorf("joinNonEmpty mismatch (-want +got):\n%s", diff)
	}
}
