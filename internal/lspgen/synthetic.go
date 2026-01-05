// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"strings"
)

type unionCandidate struct {
	Name  string
	Items []metaType
}

func (g *generator) collectSyntheticTypes() error {
	for _, s := range g.meta.Structures {
		structName := goName(s.Name)
		if !g.isReachable(structName) {
			continue
		}
		for _, prop := range s.Properties {
			g.maybeRegisterPropertyAlias(structName, prop)
			if err := g.visitType(prop.Type); err != nil {
				return err
			}
		}

		for _, ext := range s.Extends {
			if err := g.visitType(ext); err != nil {
				return err
			}
		}

		for _, mixin := range s.Mixins {
			if err := g.visitType(mixin); err != nil {
				return err
			}
		}
	}

	for _, alias := range g.meta.TypeAliases {
		aliasName := goName(alias.Name)
		if aliasName == "LSPAny" || aliasName == "LSPArray" || aliasName == "LSPObject" {
			continue
		}
		if !g.isReachable(aliasName) {
			continue
		}
		if alias.Type.Kind == metaTypeOr {
			if err := g.visitTypeWithOptions(alias.Type, false); err != nil {
				return err
			}
			continue
		}
		if err := g.visitType(alias.Type); err != nil {
			return err
		}
	}

	for _, enum := range g.meta.Enumerations {
		if !g.isReachable(goName(enum.Name)) {
			continue
		}
		if err := g.visitType(enum.Type); err != nil {
			return err
		}
	}

	for _, req := range g.meta.Requests {
		if req.Params != nil {
			if err := g.visitType(*req.Params); err != nil {
				return err
			}
		}
		if req.Result != nil {
			if err := g.visitType(*req.Result); err != nil {
				return err
			}
		}
		if req.PartialResult != nil {
			if err := g.visitType(*req.PartialResult); err != nil {
				return err
			}
		}
		if req.RegistrationOptions != nil {
			if req.RegistrationOptions.Kind != metaTypeReference {
				if err := g.registerRegistrationOptions(req.TypeName, *req.RegistrationOptions); err != nil {
					return err
				}
			}
			if err := g.visitType(*req.RegistrationOptions); err != nil {
				return err
			}
		}
	}

	for _, note := range g.meta.Notifications {
		if note.Params != nil {
			if err := g.visitType(*note.Params); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *generator) registerRegistrationOptions(typeName string, options metaType) error {
	base := strings.TrimSuffix(goName(typeName), "Request")
	if base == "" {
		return nil
	}

	name := base + "RegistrationOptions"
	if _, exists := g.registry.structs[name]; exists {
		return nil
	}
	if _, exists := g.syntheticStructs[name]; exists {
		return nil
	}

	if options.Kind != metaTypeAnd {
		return nil
	}
	if len(options.Items) == 0 {
		return nil
	}

	g.syntheticStructs[name] = structure{
		Name:   name,
		Mixins: options.Items,
	}

	return nil
}

func (g *generator) maybeRegisterPropertyAlias(structName string, prop property) {
	if prop.Type.Kind != metaTypeReference {
		return
	}
	if goName(prop.Type.Name) != "LSPAny" {
		return
	}

	var aliasName string
	switch prop.Name {
	case "data":
		aliasName = structName + "Data"

	case "initializationOptions":
		aliasName = "InitializationOptions"

	case "registerOptions":
		aliasName = "RegisterOptions"

	default:
		return
	}

	if _, exists := g.syntheticAliases[aliasName]; exists {
		return
	}

	alias := typeAlias{
		Name:          aliasName,
		Type:          prop.Type,
		Documentation: prop.Documentation,
		Since:         prop.Since,
		Deprecated:    prop.Deprecated,
		Proposed:      prop.Proposed,
	}
	g.syntheticAliases[aliasName] = alias
}

func (g *generator) visitType(t metaType) error {
	return g.visitTypeWithOptions(t, true)
}

func (g *generator) visitTypeWithOptions(t metaType, registerUnion bool) error {
	switch t.Kind {
	case metaTypeBase:
		switch t.Name {
		case "null":
			g.needsNull = true
		case "DocumentUri", "URI":
			g.needsURI = true
		}

	case metaTypeOr:
		if g.isLSPAnyOrNull(t.Items) {
			return nil
		}
		if registerUnion {
			union, err := g.registerUnionInline(t)
			if err != nil {
				return err
			}
			for _, item := range union.Items {
				if err := g.visitTypeWithOptions(item, true); err != nil {
					return err
				}
			}
			return nil
		}
		for _, item := range t.Items {
			if err := g.visitTypeWithOptions(item, true); err != nil {
				return err
			}
		}
		return nil

	case metaTypeArray:
		if t.Element != nil {
			return g.visitTypeWithOptions(*t.Element, true)
		}

	case metaTypeMap:
		if t.Key != nil {
			if err := g.visitTypeWithOptions(*t.Key, true); err != nil {
				return err
			}
		}
		if t.Value != nil {
			return g.visitTypeWithOptions(*t.Value, true)
		}

	case metaTypeLiteral:
		if t.Literal != nil {
			if len(t.Literal.Properties) == 0 {
				g.needsEmptyObj = true
			} else {
				if _, err := g.registerLiteralStruct(t.Literal.Properties); err != nil {
					return err
				}
			}
			for _, prop := range t.Literal.Properties {
				if err := g.visitTypeWithOptions(prop.Type, true); err != nil {
					return err
				}
			}
		}

	case metaTypeStringLiteral:
		if err := g.registerStringLiteral(t.StringLiteral); err != nil {
			return err
		}

	case metaTypeTuple:
		g.needsTuple = true
		for _, item := range t.Items {
			if err := g.visitTypeWithOptions(item, true); err != nil {
				return err
			}
		}

	case metaTypeAnd:
		for _, item := range t.Items {
			if err := g.visitTypeWithOptions(item, true); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *generator) registerLiteralStruct(props []property) (string, error) {
	if len(props) == 0 {
		g.needsEmptyObj = true
		return "EmptyObject", nil
	}

	signature := g.literalSignature(props)
	if name, ok := g.literalNames[signature]; ok {
		return name, nil
	}

	base := literalStructNameFromSignature(signature)
	name := base
	for i := 1; ; i++ {
		conflict := false
		for sig, existing := range g.literalNames {
			if existing == name && sig != signature {
				conflict = true
				break
			}
		}
		if !conflict {
			break
		}
		name = fmt.Sprintf("%s%d", base, i+1)
	}

	propsCopy := append([]property(nil), props...)
	g.syntheticStructs[name] = structure{
		Name:       name,
		Properties: propsCopy,
	}
	g.literalNames[signature] = name

	return name, nil
}

func (g *generator) literalSignature(props []property) string {
	var b strings.Builder
	for _, prop := range props {
		b.WriteString(prop.Name)
		b.WriteByte(':')
		b.WriteString(g.metaTypeSignature(prop.Type))
		if prop.Optional {
			b.WriteByte('?')
		}
		b.WriteByte(';')
	}

	return b.String()
}

func (g *generator) metaTypeSignature(t metaType) string {
	switch t.Kind {
	case metaTypeBase:
		return "base:" + t.Name

	case metaTypeReference:
		return "ref:" + goName(t.Name)

	case metaTypeArray:
		if t.Element == nil {
			return "array:nil"
		}
		return "array:" + g.metaTypeSignature(*t.Element)

	case metaTypeMap:
		key := "nil"
		if t.Key != nil {
			key = g.metaTypeSignature(*t.Key)
		}
		val := "nil"
		if t.Value != nil {
			val = g.metaTypeSignature(*t.Value)
		}
		return "map[" + key + "]" + val

	case metaTypeOr:
		var b strings.Builder
		b.WriteString("or(")
		for _, item := range t.Items {
			b.WriteString(g.metaTypeSignature(item))
			b.WriteByte('|')
		}
		b.WriteByte(')')
		return b.String()

	case metaTypeAnd:
		var b strings.Builder
		b.WriteString("and(")
		for _, item := range t.Items {
			b.WriteString(g.metaTypeSignature(item))
			b.WriteByte('&')
		}
		b.WriteByte(')')
		return b.String()

	case metaTypeTuple:
		var b strings.Builder
		b.WriteString("tuple(")
		for _, item := range t.Items {
			b.WriteString(g.metaTypeSignature(item))
			b.WriteByte(',')
		}
		b.WriteByte(')')
		return b.String()

	case metaTypeLiteral:
		if t.Literal == nil {
			return "literal:nil"
		}
		return "literal:" + g.literalSignature(t.Literal.Properties)

	case metaTypeStringLiteral:
		return "stringLiteral:" + t.StringLiteral

	default:
		return "unknown"
	}
}

func literalStructNameFromSignature(signature string) string {
	const prefix = "LiteralObject"
	if signature == "" {
		return prefix
	}
	h := fnv32a(signature)

	return fmt.Sprintf("%s%08x", prefix, h)
}

func fnv32a(value string) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	hash := uint32(offset32)
	for i := 0; i < len(value); i++ {
		hash ^= uint32(value[i])
		hash *= prime32
	}
	return hash
}

func (g *generator) registerStringLiteral(value string) error {
	name := stringLiteralTypeName(value)
	if existing, ok := g.stringLiterals[name]; ok {
		if existing != value {
			return fmt.Errorf("string literal name %s has conflicting values %q and %q", name, existing, value)
		}
		return nil
	}
	g.stringLiterals[name] = value

	return nil
}

func (g *generator) registerUnionInline(t metaType) (unionDef, error) {
	name, items, err := g.unionNameAndItems(t)
	if err != nil {
		return unionDef{}, err
	}

	return g.registerUnionType(name, items), nil
}

func (g *generator) registerUnionType(name string, items []metaType) unionDef {
	if existing, ok := g.unionDefs[name]; ok {
		existingLabels := g.unionItemLabels(existing.Items)
		newLabels := g.unionItemLabels(items)
		if !slicesEqual(existingLabels, newLabels) {
			g.warnings = append(g.warnings, fmt.Sprintf("union %s has conflicting items", name))
		}
		return existing
	}
	union := unionDef{
		Name:  name,
		Items: append([]metaType(nil), items...),
	}
	g.unionDefs[name] = union

	return union
}

func (g *generator) unionNameAndItems(t metaType) (string, []metaType, error) {
	candidates := g.unionCandidates(t)

	if len(candidates) == 0 {
		return "", nil, fmt.Errorf("union has no candidates")
	}
	for _, candidate := range candidates {
		if _, ok := g.structMapSet[candidate.Name]; ok {
			return candidate.Name, candidate.Items, nil
		}
	}

	return candidates[0].Name, candidates[0].Items, nil
}

func (g *generator) unionCandidates(t metaType) []unionCandidate {
	lists := g.expandUnionItems(t.Items)
	seen := make(map[string]struct{})
	candidates := make([]unionCandidate, 0, len(lists))
	for _, items := range lists {
		name := g.unionLabelFromItems(items)
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		candidates = append(candidates, unionCandidate{Name: name, Items: items})
	}

	return candidates
}

func (g *generator) expandUnionItems(items []metaType) [][]metaType {
	result := [][]metaType{{}}
	for _, item := range items {
		switch item.Kind {
		case metaTypeOr:
			expanded := g.expandUnionItems(item.Items)
			combined := make([][]metaType, 0, len(result)*len(expanded))
			for _, base := range result {
				for _, extra := range expanded {
					combined = append(combined, append(append([]metaType(nil), base...), extra...))
				}
			}
			result = combined

		case metaTypeReference:
			refName := goName(item.Name)
			alias, ok := g.registry.aliases[refName]
			if ok && alias.Type.Kind == metaTypeOr {
				expanded := g.expandUnionItems(alias.Type.Items)
				combined := make([][]metaType, 0, len(result)*(len(expanded)+1))
				for _, base := range result {
					combined = append(combined, append(append([]metaType(nil), base...), item))
					for _, extra := range expanded {
						combined = append(combined, append(append([]metaType(nil), base...), extra...))
					}
				}
				result = combined
				break
			}
			for i := range result {
				result[i] = append(result[i], item)
			}

		default:
			for i := range result {
				result[i] = append(result[i], item)
			}
		}
	}

	return result
}

func (g *generator) unionLabelFromItems(items []metaType) string {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, g.typeLabel(item))
	}

	nonNull := make([]string, 0, len(labels))
	for _, name := range labels {
		if name != "Null" {
			nonNull = append(nonNull, name)
		}
	}
	if len(nonNull) > 1 {
		prefix := commonPrefixWords(nonNull)
		if len(prefix) > 0 {
			trimmed := make([]string, 0, len(labels))
			first := true
			for _, name := range labels {
				if name == "Null" {
					trimmed = append(trimmed, name)
					continue
				}
				if first {
					trimmed = append(trimmed, name)
					first = false
					continue
				}
				trimmed = append(trimmed, trimPrefixWords(name, prefix))
			}
			labels = trimmed
		}
	}

	return strings.Join(labels, "Or")
}

func (g *generator) typeLabel(t metaType) string {
	switch t.Kind {
	case metaTypeBase:
		return baseTypeLabel(t.Name)

	case metaTypeReference:
		return goName(t.Name)

	case metaTypeArray:
		if t.Element == nil {
			return ""
		}
		elemLabel := g.typeLabel(*t.Element)
		if t.Element.Kind == metaTypeOr {
			return elemLabel + "Array"
		}
		return pluralize(elemLabel)

	case metaTypeStringLiteral:
		return stringLiteralTypeName(t.StringLiteral)

	case metaTypeLiteral:
		if t.Literal == nil || len(t.Literal.Properties) == 0 {
			return "EmptyObject"
		}
		signature := g.literalSignature(t.Literal.Properties)
		return literalStructNameFromSignature(signature)

	case metaTypeTuple:
		return "Tuple"

	case metaTypeAnd:
		labels := make([]string, 0, len(t.Items))
		for _, item := range t.Items {
			labels = append(labels, g.typeLabel(item))
		}
		return strings.Join(labels, "And")

	case metaTypeOr:
		return g.unionLabelFromItems(t.Items)

	default:
		return ""
	}
}

func baseTypeLabel(name string) string {
	switch name {
	case "string":
		return "String"
	case "boolean":
		return "Boolean"
	case "integer":
		return "Integer"
	case "uinteger":
		return "Uinteger"
	case "decimal":
		return "Decimal"
	case "null":
		return "Null"
	case "DocumentUri":
		return "DocumentURI"
	case "URI":
		return "URI"
	default:
		return goName(name)
	}
}

func stringLiteralTypeName(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return (r < '0' || r > '9') && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z')
	})

	var b strings.Builder
	b.WriteString("StringLiteral")
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(part[1:])
		}
	}
	if b.Len() == len("StringLiteral") {
		b.WriteString("Value")
	}

	return b.String()
}

func commonPrefixWords(names []string) []string {
	if len(names) == 0 {
		return nil
	}

	prefix := splitWords(names[0])
	for _, name := range names[1:] {
		words := splitWords(name)
		maxWord := min(len(words), len(prefix))
		match := 0
		for match < maxWord {
			if words[match] != prefix[match] {
				break
			}
			match++
		}
		prefix = prefix[:match]
		if len(prefix) == 0 {
			return nil
		}
	}

	return prefix
}

func trimPrefixWords(name string, prefix []string) string {
	if len(prefix) == 0 {
		return name
	}

	words := splitWords(name)
	if len(words) < len(prefix) {
		return name
	}
	for i := range prefix {
		if words[i] != prefix[i] {
			return name
		}
	}
	trimmed := words[len(prefix):]
	if len(trimmed) == 0 {
		return name
	}

	return strings.Join(trimmed, "")
}

func (g *generator) unionItemLabels(items []metaType) []string {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, g.typeLabel(item))
	}

	return labels
}

func slicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}

	return true
}
