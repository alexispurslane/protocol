// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"strings"
)

type jsonKindMask uint8

const (
	kindMaskNull jsonKindMask = 1 << iota
	kindMaskBool
	kindMaskString
	kindMaskNumber
	kindMaskObject
	kindMaskArray
)

type unionVariant struct {
	FieldName      string
	TypeName       string
	Meta           metaType
	Mask           jsonKindMask
	Discriminators []discriminator
}

type typeOptions struct {
	resolved bool
}

func (g *generator) goType(t metaType, opts typeOptions) (string, error) {
	switch t.Kind {
	case metaTypeBase:
		return g.baseGoType(t.Name)

	case metaTypeReference:
		name := goName(t.Name)
		if opts.resolved {
			resolved := "Resolved" + name
			if _, ok := g.structMapSet[resolved]; ok {
				name = resolved
			}
		}
		if override := referenceOverrideType(name); override != "" {
			return override, nil
		}
		return name, nil

	case metaTypeArray:
		if t.Element == nil {
			return "[]any", fmt.Errorf("array type missing element")
		}
		elem, err := g.goType(*t.Element, opts)
		if err != nil {
			return "", err
		}
		return "[]" + elem, nil

	case metaTypeMap:
		if t.Value == nil {
			return "map[string]any", fmt.Errorf("map type missing value")
		}
		val, err := g.goType(*t.Value, opts)
		if err != nil {
			return "", err
		}
		return "map[string]" + val, nil

	case metaTypeOr:
		if g.isLSPAnyOrNull(t.Items) {
			return "any", nil
		}
		name, items, err := g.unionNameAndItems(t)
		if err != nil {
			return "", err
		}
		g.registerUnionType(name, items)
		return name, nil

	case metaTypeAnd:
		name := g.typeLabel(t)
		if name == "" {
			return "", fmt.Errorf("and type missing name")
		}
		if _, ok := g.registry.structs[name]; !ok {
			if _, ok := g.syntheticStructs[name]; !ok {
				items := append([]metaType(nil), t.Items...)
				g.syntheticStructs[name] = structure{Name: name, Mixins: items}
			}
		}
		return name, nil

	case metaTypeTuple:
		g.needsTuple = true
		return "Tuple", nil

	case metaTypeLiteral:
		if t.Literal == nil || len(t.Literal.Properties) == 0 {
			g.needsEmptyObj = true
			return "EmptyObject", nil
		}
		name, err := g.registerLiteralStruct(t.Literal.Properties)
		if err != nil {
			return "", err
		}
		return name, nil

	case metaTypeStringLiteral:
		if err := g.registerStringLiteral(t.StringLiteral); err != nil {
			return "", err
		}
		return stringLiteralTypeName(t.StringLiteral), nil

	default:
		return "", fmt.Errorf("unsupported metaType kind %q", t.Kind)
	}
}

func (g *generator) baseGoType(name string) (string, error) {
	switch name {
	case "string":
		return "string", nil
	case "boolean":
		return "bool", nil
	case "integer":
		return "int32", nil
	case "uinteger":
		return "uint32", nil
	case "decimal":
		return "float64", nil
	case "null":
		g.needsNull = true
		return "Null", nil
	case "DocumentUri":
		g.needsURI = true
		return "DocumentURI", nil
	case "URI":
		g.needsURI = true
		return "URI", nil
	default:
		return goName(name), nil
	}
}

func (g *generator) shouldPointer(t metaType) bool {
	switch t.Kind {
	case metaTypeArray, metaTypeMap, metaTypeTuple:
		return false

	case metaTypeBase:
		if g.isPrimitiveGoType(t) {
			return false
		}
		return t.Name != "null"

	case metaTypeLiteral:
		return t.Literal != nil && len(t.Literal.Properties) > 0

	case metaTypeOr:
		return true

	case metaTypeStringLiteral:
		return true

	case metaTypeReference:
		name := goName(t.Name)
		if aliasOverrideType(name) != "" {
			return false
		}
		if alias, ok := g.registry.aliases[name]; ok {
			return g.shouldPointer(alias.Type)
		}
		if _, ok := g.registry.enums[name]; ok {
			return true
		}
		if _, ok := g.registry.structs[name]; ok {
			return true
		}
		if _, ok := g.syntheticStructs[name]; ok {
			return true
		}
		if _, ok := g.unionDefs[name]; ok {
			return true
		}
		return true

	case metaTypeAnd:
		return true

	default:
		return true
	}
}

func (g *generator) isPrimitiveGoType(t metaType) bool {
	if t.Kind != metaTypeBase {
		return false
	}
	switch t.Name {
	case "string", "boolean", "integer", "uinteger", "decimal":
		return true
	default:
		return false
	}
}

func referenceOverrideType(name string) string {
	switch name {
	case "LSPAny", "LSPArray", "LSPObject":
		return aliasOverrideType(name)
	default:
		return ""
	}
}

func (g *generator) isLSPAnyOrNull(items []metaType) bool {
	hasLSPAny := false
	hasNull := false
	for _, item := range items {
		switch item.Kind {
		case metaTypeReference:
			if goName(item.Name) != "LSPAny" {
				return false
			}
			hasLSPAny = true
		case metaTypeBase:
			if item.Name != "null" {
				return false
			}
			hasNull = true
		default:
			return false
		}
	}
	return hasLSPAny && hasNull
}

func (g *generator) propertyGoType(structName string, prop property, opts typeOptions) (string, error) {
	if prop.Type.Kind == metaTypeReference && goName(prop.Type.Name) == "LSPAny" {
		switch prop.Name {
		case "data":
			return structName + "Data", nil
		case "initializationOptions":
			return "InitializationOptions", nil
		case "registerOptions":
			return "RegisterOptions", nil
		}
	}

	return g.goType(prop.Type, opts)
}

func (g *generator) unionVariants(def unionDef) ([]unionVariant, error) {
	variants := make([]unionVariant, 0, len(def.Items))
	for _, item := range def.Items {
		label := g.typeLabel(item)
		if label == "" {
			return nil, fmt.Errorf("union %s has empty label", def.Name)
		}
		typeName, err := g.goType(item, typeOptions{})
		if err != nil {
			return nil, err
		}

		variants = append(variants, unionVariant{
			FieldName:      label,
			TypeName:       typeName,
			Meta:           item,
			Mask:           g.kindMask(item),
			Discriminators: g.discriminatorsForType(item),
		})
	}

	return ensureUniqueUnionFieldNames(variants), nil
}

func ensureUniqueUnionFieldNames(variants []unionVariant) []unionVariant {
	counts := make(map[string]int)
	for i := range variants {
		name := variants[i].FieldName
		counts[name]++
		if counts[name] > 1 {
			variants[i].FieldName = fmt.Sprintf("%s%d", name, counts[name])
		}
	}

	return variants
}

func (g *generator) kindMask(t metaType) jsonKindMask {
	switch t.Kind {
	case metaTypeBase:
		switch t.Name {
		case "null":
			return kindMaskNull

		case "boolean":
			return kindMaskBool

		case "string", "DocumentUri", "URI":
			return kindMaskString

		case "integer", "uinteger", "decimal":
			return kindMaskNumber

		default:
			return kindMaskObject
		}

	case metaTypeStringLiteral:
		return kindMaskString

	case metaTypeLiteral:
		return kindMaskObject

	case metaTypeArray, metaTypeTuple:
		return kindMaskArray

	case metaTypeMap:
		return kindMaskObject

	case metaTypeAnd:
		return kindMaskObject

	case metaTypeReference:
		name := goName(t.Name)
		if alias, ok := g.registry.aliases[name]; ok {
			if alias.Type.Kind == metaTypeOr {
				return kindMaskNull | kindMaskBool | kindMaskString | kindMaskNumber | kindMaskObject | kindMaskArray
			}
			return g.kindMask(alias.Type)
		}
		if _, ok := g.unionDefs[name]; ok {
			return kindMaskNull | kindMaskBool | kindMaskString | kindMaskNumber | kindMaskObject | kindMaskArray
		}
		if enum, ok := g.registry.enums[name]; ok {
			return g.kindMask(enum.Type)
		}
		if _, ok := g.registry.structs[name]; ok {
			return kindMaskObject
		}
		if _, ok := g.syntheticStructs[name]; ok {
			return kindMaskObject
		}
		return kindMaskNull | kindMaskBool | kindMaskString | kindMaskNumber | kindMaskObject | kindMaskArray

	case metaTypeOr:
		return kindMaskNull | kindMaskBool | kindMaskString | kindMaskNumber | kindMaskObject | kindMaskArray

	default:
		return kindMaskNull | kindMaskBool | kindMaskString | kindMaskNumber | kindMaskObject | kindMaskArray
	}
}

func (g *generator) discriminatorsForType(t metaType) []discriminator {
	switch t.Kind {
	case metaTypeReference:
		name := goName(t.Name)
		if discs := g.registry.discriminators[name]; len(discs) > 0 {
			return discs
		}
		if synth, ok := g.syntheticStructs[name]; ok {
			return g.discriminatorsFromProperties(synth.Properties)
		}
		if alias, ok := g.registry.aliases[name]; ok {
			return g.discriminatorsForType(alias.Type)
		}

	case metaTypeAnd:
		var all []discriminator
		for _, item := range t.Items {
			all = append(all, g.discriminatorsForType(item)...)
		}
		return all
	}

	return nil
}

func (g *generator) discriminatorsFromProperties(props []property) []discriminator {
	var discs []discriminator
	for _, prop := range props {
		if prop.Type.Kind != metaTypeStringLiteral {
			continue
		}
		discs = append(discs, discriminator{
			JSONName: prop.Name,
			Value:    prop.Type.StringLiteral,
		})
	}

	return discs
}

func docLines(doc, since, deprecated string, proposed bool) []string {
	lines, state := normalizeDocTags(splitDocLines(doc))
	if since != "" && !state.hasSince {
		sinceLines, sinceProposed := formatSinceLines(since)
		lines = append(lines, sinceLines...)
		if sinceProposed {
			state.hasProposed = true
		}
	}
	if deprecated != "" && !state.hasDeprecated {
		lines = append(lines, formatDeprecatedLines(deprecated)...)
	}
	if proposed && !state.hasProposed {
		lines = append(lines, "Proposed")
	}
	for i, line := range lines {
		lines[i] = ensurePeriod(line)
	}

	return lines
}

type docTagState struct {
	hasSince      bool
	hasDeprecated bool
	hasProposed   bool
}

func normalizeDocTags(lines []string) ([]string, docTagState) {
	if len(lines) == 0 {
		return nil, docTagState{}
	}

	out := make([]string, 0, len(lines))
	state := docTagState{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "@since"):
			if state.hasSince {
				continue
			}
			state.hasSince = true
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "@since"))
			formatted, proposed := formatSinceLine(value)
			if proposed {
				state.hasProposed = true
			}
			out = append(out, formatted)

		case strings.HasPrefix(trimmed, "@deprecated"):
			if state.hasDeprecated {
				continue
			}
			state.hasDeprecated = true
			out = append(out, line)

		case strings.HasPrefix(trimmed, "@proposed"):
			if state.hasProposed {
				continue
			}
			state.hasProposed = true
			out = append(out, "Proposed")

		default:
			out = append(out, line)
		}
	}

	return out, state
}

func formatSinceLines(value string) ([]string, bool) {
	if value == "" {
		return nil, false
	}
	parts := splitDocLines(value)
	if len(parts) == 0 {
		return nil, false
	}

	first, proposed := formatSinceLine(parts[0])
	lines := make([]string, 0, len(parts))
	lines = append(lines, first)
	lines = append(lines, parts[1:]...)

	return lines, proposed
}

func formatSinceLine(value string) (string, bool) {
	if value == "" {
		return "Since", false
	}
	base := strings.TrimSpace(value)
	proposed := false
	lower := strings.ToLower(base)
	const marker = " - proposed"
	switch {
	case strings.HasSuffix(lower, marker+"."):
		base = strings.TrimSpace(base[:len(base)-len(marker+".")])
		proposed = true

	case strings.HasSuffix(lower, marker):
		base = strings.TrimSpace(base[:len(base)-len(marker)])
		proposed = true
	}

	line := "Since"
	if base != "" {
		line = "Since " + base
	}
	if proposed {
		line += ", Proposed"
	}

	return line, proposed
}

func formatDeprecatedLines(value string) []string {
	if value == "" {
		return nil
	}

	parts := splitDocLines(value)
	if len(parts) == 0 {
		return nil
	}

	lines := make([]string, 0, len(parts))
	for i, part := range parts {
		if i == 0 {
			lines = append(lines, formatDeprecatedLine(part))
			continue
		}
		lines = append(lines, part)
	}

	return lines
}

func formatDeprecatedLine(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "Deprecated"
	}
	return "Deprecated. " + trimmed
}

func splitDocLines(doc string) []string {
	if doc == "" {
		return nil
	}
	parts := strings.Split(doc, "\n")
	lines := make([]string, 0, len(parts))
	for _, line := range parts {
		lines = append(lines, strings.TrimRight(line, " \t"))
	}

	return lines
}

func ensurePeriod(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	last := trimmed[len(trimmed)-1]
	if last == '.' || last == '!' || last == '?' {
		return line
	}
	return line + "."
}

func commentLines(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			out = append(out, "//")
			continue
		}
		out = append(out, "// "+line)
	}
	return out
}

func formatConstName(typeName, valueName string) string {
	return typeName + goName(valueName)
}

func joinNonEmpty(lines []string) string {
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
