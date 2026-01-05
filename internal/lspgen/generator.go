// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"sort"
	"strings"
)

type generator struct {
	cfg              config
	meta             metaModel
	structMap        []structMapEntry
	structMapSet     map[string]struct{}
	structMapByName  map[string]string
	structMapOrder   map[string]int
	registry         *registry
	unionDefs        map[string]unionDef
	stringLiterals   map[string]string
	needsTuple       bool
	needsEmptyObj    bool
	needsNull        bool
	needsURI         bool
	reachable        map[string]struct{}
	syntheticAliases map[string]typeAlias
	syntheticStructs map[string]structure
	literalNames     map[string]string
	warnings         []string
}

type registry struct {
	structs        map[string]structure
	enums          map[string]enumeration
	aliases        map[string]typeAlias
	discriminators map[string][]discriminator
}

type discriminator struct {
	JSONName string
	Value    string
}

type unionDef struct {
	Name  string
	Items []metaType
}

type typeDefKind int

const (
	typeDefStruct typeDefKind = iota
	typeDefResolvedStruct
	typeDefEnum
	typeDefAlias
	typeDefUnion
	typeDefStringLiteral
	typeDefTuple
	typeDefEmptyObject
	typeDefNull
)

type typeDef struct {
	Name         string
	Kind         typeDefKind
	Struct       *structure
	Enum         *enumeration
	Alias        *typeAlias
	Union        *unionDef
	StringValue  string
	ResolvedFrom string
}

func newGenerator(model metaModel, entries []structMapEntry, cfg config) (*generator, error) {
	gen := &generator{
		cfg:              cfg,
		meta:             model,
		structMap:        entries,
		structMapSet:     make(map[string]struct{}, len(entries)),
		structMapByName:  make(map[string]string, len(entries)),
		structMapOrder:   make(map[string]int, len(entries)),
		unionDefs:        make(map[string]unionDef),
		stringLiterals:   make(map[string]string),
		syntheticAliases: make(map[string]typeAlias),
		syntheticStructs: make(map[string]structure),
		literalNames:     make(map[string]string),
	}

	for i, entry := range entries {
		gen.structMapSet[entry.Name] = struct{}{}
		gen.structMapByName[entry.Name] = entry.File
		gen.structMapOrder[entry.Name] = i
	}

	reg, err := buildRegistry(model)
	if err != nil {
		return nil, err
	}
	gen.registry = reg
	gen.reachable = gen.computeReachable()

	if err := gen.collectSyntheticTypes(); err != nil {
		return nil, err
	}

	return gen, nil
}

func buildRegistry(model metaModel) (*registry, error) {
	reg := &registry{
		structs:        make(map[string]structure, len(model.Structures)),
		enums:          make(map[string]enumeration, len(model.Enumerations)),
		aliases:        make(map[string]typeAlias, len(model.TypeAliases)),
		discriminators: make(map[string][]discriminator),
	}

	for _, s := range model.Structures {
		name := goName(s.Name)
		if _, exists := reg.structs[name]; exists {
			return nil, fmt.Errorf("duplicate structure name %q", name)
		}
		reg.structs[name] = s

		for _, prop := range s.Properties {
			if prop.Type.Kind != metaTypeStringLiteral {
				continue
			}
			reg.discriminators[name] = append(reg.discriminators[name], discriminator{
				JSONName: prop.Name,
				Value:    prop.Type.StringLiteral,
			})
		}
	}

	for _, e := range model.Enumerations {
		name := goName(e.Name)
		if _, exists := reg.enums[name]; exists {
			return nil, fmt.Errorf("duplicate enumeration name %q", name)
		}
		reg.enums[name] = e
	}

	for _, a := range model.TypeAliases {
		name := goName(a.Name)
		if _, exists := reg.aliases[name]; exists {
			return nil, fmt.Errorf("duplicate type alias name %q", name)
		}
		reg.aliases[name] = a
	}

	return reg, nil
}

func (g *generator) Generate() (map[string][]byte, error) {
	defs, err := g.collectTypeDefs()
	if err != nil {
		return nil, err
	}

	files, err := g.renderFiles(defs)
	if err != nil {
		return nil, err
	}

	if g.cfg.strict {
		if err := g.validateStrict(defs); err != nil {
			return nil, err
		}
	}

	return files, nil
}

func (g *generator) validateStrict(defs map[string]typeDef) error {
	var missing []string
	for _, entry := range g.structMap {
		if isSkippedType(entry.Name) || !g.isReachable(entry.Name) {
			continue
		}
		if _, ok := defs[entry.Name]; !ok {
			missing = append(missing, entry.Name)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("struct-map entries without generated types: %s", strings.Join(missing, ", "))
	}

	return nil
}

func isSkippedType(name string) bool {
	switch name {
	case "LSPAny", "LSPArray", "LSPObject", "LSPAnyOrNull":
		return true
	default:
		return false
	}
}

func (g *generator) collectTypeDefs() (map[string]typeDef, error) {
	defs := make(map[string]typeDef)

	for name, s := range g.registry.structs {
		if !g.isReachable(name) {
			continue
		}
		structCopy := s
		defs[name] = typeDef{
			Name:   name,
			Kind:   typeDefStruct,
			Struct: &structCopy,
		}
	}

	for name, s := range g.syntheticStructs {
		structCopy := s
		if _, exists := defs[name]; exists {
			continue
		}
		defs[name] = typeDef{
			Name:   name,
			Kind:   typeDefStruct,
			Struct: &structCopy,
		}
	}

	for name, e := range g.registry.enums {
		if !g.isReachable(name) {
			continue
		}
		enumCopy := e
		defs[name] = typeDef{
			Name: name,
			Kind: typeDefEnum,
			Enum: &enumCopy,
		}
	}

	for name, a := range g.registry.aliases {
		if !g.isReachable(name) {
			continue
		}
		aliasCopy := a
		if name == "LSPAny" || name == "LSPArray" || name == "LSPObject" {
			continue
		}
		if aliasOverrideType(name) != "" {
			defs[name] = typeDef{
				Name:  name,
				Kind:  typeDefAlias,
				Alias: &aliasCopy,
			}
			continue
		}
		if a.Type.Kind == metaTypeOr {
			union := g.registerUnionType(name, a.Type.Items)
			defs[name] = typeDef{
				Name:  name,
				Kind:  typeDefUnion,
				Union: &union,
			}
			continue
		}
		if a.Type.Kind == metaTypeTuple {
			g.needsTuple = true
			defs[name] = typeDef{
				Name:  name,
				Kind:  typeDefAlias,
				Alias: &aliasCopy,
			}
			continue
		}
		defs[name] = typeDef{
			Name:  name,
			Kind:  typeDefAlias,
			Alias: &aliasCopy,
		}
	}

	for name, alias := range g.syntheticAliases {
		aliasCopy := alias
		if _, exists := defs[name]; exists {
			continue
		}
		defs[name] = typeDef{
			Name:  name,
			Kind:  typeDefAlias,
			Alias: &aliasCopy,
		}
	}

	for name, union := range g.unionDefs {
		if _, exists := defs[name]; exists {
			continue
		}
		unionCopy := union
		defs[name] = typeDef{
			Name:  name,
			Kind:  typeDefUnion,
			Union: &unionCopy,
		}
	}

	for name, value := range g.stringLiterals {
		if _, exists := defs[name]; exists {
			continue
		}
		defs[name] = typeDef{
			Name:        name,
			Kind:        typeDefStringLiteral,
			StringValue: value,
		}
	}

	if g.needsTuple {
		if _, exists := defs["Tuple"]; !exists {
			defs["Tuple"] = typeDef{Name: "Tuple", Kind: typeDefTuple}
		}
	}

	if g.needsEmptyObj {
		if _, exists := defs["EmptyObject"]; !exists {
			defs["EmptyObject"] = typeDef{Name: "EmptyObject", Kind: typeDefEmptyObject}
		}
	}

	if g.needsNull {
		if _, exists := defs["Null"]; !exists {
			defs["Null"] = typeDef{Name: "Null", Kind: typeDefNull}
		}
	}

	if g.needsURI {
		if _, exists := defs["DocumentURI"]; !exists {
			defs["DocumentURI"] = typeDef{
				Name: "DocumentURI",
				Kind: typeDefAlias,
				Alias: &typeAlias{
					Name:          "DocumentURI",
					Documentation: "DocumentURI represents the URI of a document.",
				},
			}
		}
		if _, exists := defs["URI"]; !exists {
			defs["URI"] = typeDef{
				Name: "URI",
				Kind: typeDefAlias,
				Alias: &typeAlias{
					Name:          "URI",
					Documentation: "URI is a tagging interface for non-document URIs.",
				},
			}
		}
	}

	for _, entry := range g.structMap {
		if !strings.HasPrefix(entry.Name, "Resolved") {
			continue
		}
		base := strings.TrimPrefix(entry.Name, "Resolved")
		baseStruct, ok := g.registry.structs[base]
		if !ok {
			g.warnings = append(g.warnings, fmt.Sprintf("resolved type %s has no base structure", entry.Name))
			continue
		}
		if _, exists := defs[entry.Name]; exists {
			continue
		}
		structCopy := baseStruct
		defs[entry.Name] = typeDef{
			Name:         entry.Name,
			Kind:         typeDefResolvedStruct,
			Struct:       &structCopy,
			ResolvedFrom: base,
		}
	}

	return defs, nil
}

func aliasOverrideType(name string) string {
	switch name {
	case "LSPAny":
		return "any"
	case "LSPArray":
		return "[]any"
	case "LSPObject":
		return "map[string]any"
	case "DocumentURI", "URI":
		return "uri.URI"
	default:
		return ""
	}
}
