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

// Command lspgen generates Go types for the Language Server Protocol.
package main

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	gofumpt "mvdan.cc/gofumpt/format"
)

// Go port of microsoft/typescript-go/internal/lsp/lsproto/_generate/generate.mts.

// OrderedMap preserves insertion order for deterministic output.
type OrderedMap[V any] struct {
	keys   []string
	values map[string]V
}

func NewOrderedMap[V any]() *OrderedMap[V] {
	return &OrderedMap[V]{
		values: make(map[string]V),
	}
}

func (m *OrderedMap[V]) Set(key string, value V) {
	if _, ok := m.values[key]; !ok {
		m.keys = append(m.keys, key)
	}
	m.values[key] = value
}

func (m *OrderedMap[V]) Get(key string) (V, bool) {
	v, ok := m.values[key]
	return v, ok
}

func (m *OrderedMap[V]) Keys() []string {
	return append([]string(nil), m.keys...)
}

// Schema types.

type BaseTypes string

type MessageDirection string

type TypeKind string

const (
	kindBase           TypeKind = "base"
	kindReference      TypeKind = "reference"
	kindArray          TypeKind = "array"
	kindMap            TypeKind = "map"
	kindAnd            TypeKind = "and"
	kindOr             TypeKind = "or"
	kindTuple          TypeKind = "tuple"
	kindLiteral        TypeKind = "literal"
	kindStringLiteral  TypeKind = "stringLiteral"
	kindIntegerLiteral TypeKind = "integerLiteral"
	kindBooleanLiteral TypeKind = "booleanLiteral"
)

type Type struct {
	Kind    TypeKind `json:"kind"`
	Name    string   `json:"name,omitzero"`
	Items   []*Type  `json:"items,omitzero"`
	Key     *Type    `json:"key,omitzero"`
	Value   *Type    `json:"value,omitzero"`
	Element *Type    `json:"element,omitzero"`

	Literal      *StructureLiteral
	StringValue  *string
	IntegerValue *int
	BoolValue    *bool
}

type rawType struct {
	Kind    TypeKind       `json:"kind"`
	Name    string         `json:"name,omitzero"`
	Items   []*Type        `json:"items,omitzero"`
	Key     *Type          `json:"key,omitzero"`
	Value   jsontext.Value `json:"value,omitzero"`
	Element *Type          `json:"element,omitzero"`
}

func (t *Type) UnmarshalJSON(data []byte) error {
	var raw rawType
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	t.Kind = raw.Kind
	t.Name = raw.Name
	t.Items = raw.Items
	t.Key = raw.Key
	t.Element = raw.Element

	switch raw.Kind {
	case kindMap:
		if len(raw.Value) > 0 {
			var val Type
			if err := json.Unmarshal(raw.Value, &val); err != nil {
				return err
			}
			t.Value = &val
		}

	case kindLiteral:
		if len(raw.Value) > 0 {
			var lit StructureLiteral
			if err := json.Unmarshal(raw.Value, &lit); err != nil {
				return err
			}
			t.Literal = &lit
		}

	case kindStringLiteral:
		var v string
		if err := json.Unmarshal(raw.Value, &v); err != nil {
			return err
		}
		t.StringValue = &v

	case kindIntegerLiteral:
		var v int
		if err := json.Unmarshal(raw.Value, &v); err != nil {
			return err
		}
		t.IntegerValue = &v

	case kindBooleanLiteral:
		var v bool
		if err := json.Unmarshal(raw.Value, &v); err != nil {
			return err
		}
		t.BoolValue = &v

	default:
		if len(raw.Value) > 0 {
			// ignore
		}
	}

	return nil
}

type Property struct {
	Name          string   `json:"name"`
	Type          *Type    `json:"type"`
	Optional      bool     `json:"optional,omitzero"`
	Documentation string   `json:"documentation,omitzero"`
	Since         string   `json:"since,omitzero"`
	SinceTags     []string `json:"sinceTags,omitzero"`
	Proposed      bool     `json:"proposed,omitzero"`
	Deprecated    string   `json:"deprecated,omitzero"`
	OmitzeroValue bool     `json:"omitzeroValue,omitzero"`
}

type Structure struct {
	Name          string     `json:"name"`
	Extends       []*Type    `json:"extends,omitzero"`
	Mixins        []*Type    `json:"mixins,omitzero"`
	Properties    []Property `json:"properties"`
	Documentation string     `json:"documentation,omitzero"`
	Since         string     `json:"since,omitzero"`
	SinceTags     []string   `json:"sinceTags,omitzero"`
	Proposed      bool       `json:"proposed,omitzero"`
	Deprecated    string     `json:"deprecated,omitzero"`
}

type StructureLiteral struct {
	Properties    []Property `json:"properties"`
	Documentation string     `json:"documentation,omitzero"`
	Since         string     `json:"since,omitzero"`
	SinceTags     []string   `json:"sinceTags,omitzero"`
	Proposed      bool       `json:"proposed,omitzero"`
	Deprecated    string     `json:"deprecated,omitzero"`
}

type TypeAlias struct {
	Name string `json:"name"`
	Type *Type  `json:"type"`
}

type Enumeration struct {
	Name                 string      `json:"name"`
	Type                 *Type       `json:"type"`
	Values               []EnumEntry `json:"values"`
	SupportsCustomValues bool        `json:"supportsCustomValues,omitzero"`
	Documentation        string      `json:"documentation,omitzero"`
	Since                string      `json:"since,omitzero"`
	SinceTags            []string    `json:"sinceTags,omitzero"`
	Proposed             bool        `json:"proposed,omitzero"`
	Deprecated           string      `json:"deprecated,omitzero"`
}

type EnumEntry struct {
	Name          string `json:"name"`
	ValueRaw      any    `json:"value"`
	Documentation string `json:"documentation,omitzero"`
	Deprecated    string `json:"deprecated,omitzero"`
}

type MetaData struct {
	Version string `json:"version"`
}

type MetaModel struct {
	MetaData      MetaData       `json:"metaData"`
	Requests      []Request      `json:"requests"`
	Notifications []Notification `json:"notifications"`
	Structures    []Structure    `json:"structures"`
	Enumerations  []Enumeration  `json:"enumerations"`
	TypeAliases   []TypeAlias    `json:"typeAliases"`
}

type Request struct {
	Method              string `json:"method"`
	TypeName            string `json:"typeName,omitzero"`
	paramsRaw           jsontext.Value
	Params              *Type    `json:"-"`
	ParamsArray         []*Type  `json:"-"`
	Result              *Type    `json:"result"`
	PartialResult       *Type    `json:"partialResult,omitzero"`
	ErrorData           *Type    `json:"errorData,omitzero"`
	RegistrationMethod  string   `json:"registrationMethod,omitzero"`
	RegistrationOptions *Type    `json:"registrationOptions,omitzero"`
	MessageDirection    string   `json:"messageDirection"`
	Documentation       string   `json:"documentation,omitzero"`
	Since               string   `json:"since,omitzero"`
	SinceTags           []string `json:"sinceTags,omitzero"`
	Proposed            bool     `json:"proposed,omitzero"`
	Deprecated          string   `json:"deprecated,omitzero"`
}

type Notification struct {
	Method              string `json:"method"`
	TypeName            string `json:"typeName,omitzero"`
	paramsRaw           jsontext.Value
	Params              *Type    `json:"-"`
	ParamsArray         []*Type  `json:"-"`
	RegistrationMethod  string   `json:"registrationMethod,omitzero"`
	RegistrationOptions *Type    `json:"registrationOptions,omitzero"`
	MessageDirection    string   `json:"messageDirection"`
	Documentation       string   `json:"documentation,omitzero"`
	Since               string   `json:"since,omitzero"`
	SinceTags           []string `json:"sinceTags,omitzero"`
	Proposed            bool     `json:"proposed,omitzero"`
	Deprecated          string   `json:"deprecated,omitzero"`
}

func (r *Request) UnmarshalJSON(data []byte) error {
	var aux struct {
		Method              string         `json:"method"`
		TypeName            string         `json:"typeName,omitzero"`
		ParamsRaw           jsontext.Value `json:"params,omitzero"`
		Result              *Type          `json:"result"`
		PartialResult       *Type          `json:"partialResult,omitzero"`
		ErrorData           *Type          `json:"errorData,omitzero"`
		RegistrationMethod  string         `json:"registrationMethod,omitzero"`
		RegistrationOptions *Type          `json:"registrationOptions,omitzero"`
		MessageDirection    string         `json:"messageDirection"`
		Documentation       string         `json:"documentation,omitzero"`
		Since               string         `json:"since,omitzero"`
		SinceTags           []string       `json:"sinceTags,omitzero"`
		Proposed            bool           `json:"proposed,omitzero"`
		Deprecated          string         `json:"deprecated,omitzero"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = Request{
		Method:              aux.Method,
		TypeName:            aux.TypeName,
		paramsRaw:           aux.ParamsRaw,
		Result:              aux.Result,
		PartialResult:       aux.PartialResult,
		ErrorData:           aux.ErrorData,
		RegistrationMethod:  aux.RegistrationMethod,
		RegistrationOptions: aux.RegistrationOptions,
		MessageDirection:    aux.MessageDirection,
		Documentation:       aux.Documentation,
		Since:               aux.Since,
		SinceTags:           aux.SinceTags,
		Proposed:            aux.Proposed,
		Deprecated:          aux.Deprecated,
	}

	if len(aux.ParamsRaw) > 0 {
		switch aux.ParamsRaw[0] {
		case '[':
			var arr []*Type
			if err := json.Unmarshal(aux.ParamsRaw, &arr); err != nil {
				return err
			}
			r.ParamsArray = arr

		default:
			var t Type
			if err := json.Unmarshal(aux.ParamsRaw, &t); err != nil {
				return err
			}
			r.Params = &t
		}
	}

	return nil
}

func (n *Notification) UnmarshalJSON(data []byte) error {
	var aux struct {
		Method              string         `json:"method"`
		TypeName            string         `json:"typeName,omitzero"`
		ParamsRaw           jsontext.Value `json:"params,omitzero"`
		RegistrationMethod  string         `json:"registrationMethod,omitzero"`
		RegistrationOptions *Type          `json:"registrationOptions,omitzero"`
		MessageDirection    string         `json:"messageDirection"`
		Documentation       string         `json:"documentation,omitzero"`
		Since               string         `json:"since,omitzero"`
		SinceTags           []string       `json:"sinceTags,omitzero"`
		Proposed            bool           `json:"proposed,omitzero"`
		Deprecated          string         `json:"deprecated,omitzero"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	*n = Notification{
		Method:              aux.Method,
		TypeName:            aux.TypeName,
		paramsRaw:           aux.ParamsRaw,
		RegistrationMethod:  aux.RegistrationMethod,
		RegistrationOptions: aux.RegistrationOptions,
		MessageDirection:    aux.MessageDirection,
		Documentation:       aux.Documentation,
		Since:               aux.Since,
		SinceTags:           aux.SinceTags,
		Proposed:            aux.Proposed,
		Deprecated:          aux.Deprecated,
	}

	if len(aux.ParamsRaw) > 0 {
		switch aux.ParamsRaw[0] {
		case '[':
			var arr []*Type
			if err := json.Unmarshal(aux.ParamsRaw, &arr); err != nil {
				return err
			}
			n.ParamsArray = arr

		default:
			var t Type
			if err := json.Unmarshal(aux.ParamsRaw, &t); err != nil {
				return err
			}
			n.Params = &t
		}
	}

	return nil
}

// Custom structures to add to the model.
var customStructures = []Structure{
	{
		Name: "InitializationOptions",
		Properties: []Property{
			{
				Name: "disablePushDiagnostics",
				Type: &Type{
					Name: "boolean",
					Kind: kindBase,
				},
				Optional:      true,
				Documentation: "DisablePushDiagnostics disables automatic pushing of diagnostics to the client.",
			},
		},
		Documentation: "InitializationOptions contains user-provided initialization options.",
	},
	{
		Name: "ExportInfoMapKey",
		Properties: []Property{
			{
				Name:          "symbolName",
				Documentation: "The symbol name.",
				Type: &Type{
					Name: "string",
					Kind: kindBase,
				},
				OmitzeroValue: true,
			},
			{
				Name:          "symbolId",
				Documentation: "The symbol ID.",
				Type: &Type{
					Name: "uint64",
					Kind: kindReference,
				},
				OmitzeroValue: true,
			},
			{
				Name:          "ambientModuleName",
				Documentation: "The ambient module name.",
				Type: &Type{
					Name: "string",
					Kind: kindBase,
				},
				OmitzeroValue: true,
			},
			{
				Name:          "moduleFile",
				Documentation: "The module file path.",
				Type: &Type{
					Name: "string",
					Kind: kindBase,
				},
				OmitzeroValue: true,
			},
		},
		Documentation: "ExportInfoMapKey uniquely identifies an export for auto-import purposes.",
	},
	{
		Name: "AutoImportData",
		Properties: []Property{
			{
				Name:          "exportName",
				Documentation: "The name of the property or export in the module's symbol table. Differs from the completion name in the case of InternalSymbolName.ExportEquals and InternalSymbolName.Default.",
				Type: &Type{
					Name: "string",
					Kind: kindBase,
				},
				OmitzeroValue: true,
			},
			{
				Name:          "exportMapKey",
				Documentation: "The export map key for this auto-import.",
				Type: &Type{
					Name: "ExportInfoMapKey",
					Kind: kindReference,
				},
				OmitzeroValue: true,
			},
			{
				Name:          "moduleSpecifier",
				Documentation: "The module specifier for this auto-import.",
				Type: &Type{
					Name: "string",
					Kind: kindBase,
				},
				OmitzeroValue: true,
			},
			{
				Name:          "fileName",
				Documentation: "The file name declaring the export's module symbol, if it was an external module.",
				Type: &Type{
					Name: "string",
					Kind: kindBase,
				},
				OmitzeroValue: true,
			},
			{
				Name:          "ambientModuleName",
				Documentation: "The module name (with quotes stripped) of the export's module symbol, if it was an ambient module.",
				Type: &Type{
					Name: "string",
					Kind: kindBase,
				},
				OmitzeroValue: true,
			},
			{
				Name:          "isPackageJsonImport",
				Documentation: "True if the export was found in the package.json AutoImportProvider.",
				Type: &Type{
					Name: "boolean",
					Kind: kindBase,
				},
				OmitzeroValue: true,
			},
		},
		Documentation: "AutoImportData contains information about an auto-import suggestion.",
	},
	{
		Name: "CompletionItemData",
		Properties: []Property{
			{
				Name:          "fileName",
				Documentation: "The file name where the completion was requested.",
				Type: &Type{
					Name: "string",
					Kind: kindBase,
				},
				OmitzeroValue: true,
			},
			{
				Name:          "position",
				Documentation: "The position where the completion was requested.",
				Type: &Type{
					Name: "integer",
					Kind: kindBase,
				},
				OmitzeroValue: true,
			},
			{
				Name:          "source",
				Documentation: "Special source value for disambiguation.",
				Type: &Type{
					Name: "string",
					Kind: kindBase,
				},
				OmitzeroValue: true,
			},
			{
				Name:          "name",
				Documentation: "The name of the completion item.",
				Type: &Type{
					Name: "string",
					Kind: kindBase,
				},
				OmitzeroValue: true,
			},
			{
				Name:          "autoImport",
				Documentation: "Auto-import data for this completion item.",
				Type: &Type{
					Name: "AutoImportData",
					Kind: kindReference,
				},
				Optional: true,
			},
		},
		Documentation: "CompletionItemData is preserved on a CompletionItem between CompletionRequest and CompletionResolveRequest.",
	},
}

var (
	explicitDataStructuresOnce sync.Once
	explicitDataStructuresMap  map[string]struct{}
)

func explicitDataStructures() map[string]struct{} {
	explicitDataStructuresOnce.Do(func() {
		m := make(map[string]struct{}, len(customStructures))
		for _, s := range customStructures {
			m[s.Name] = struct{}{}
		}
		explicitDataStructuresMap = m
	})

	return explicitDataStructuresMap
}

var registerOptionsUnionType *Type

// Patch and preprocess the model.
func patchAndPreprocessModel(model *MetaModel) error {
	registrationOptionTypes := []*Type{}
	for i := range model.Requests {
		if model.Requests[i].RegistrationOptions != nil {
			registrationOptionTypes = append(registrationOptionTypes, model.Requests[i].RegistrationOptions)
		}
	}
	for i := range model.Notifications {
		if model.Notifications[i].RegistrationOptions != nil {
			registrationOptionTypes = append(registrationOptionTypes, model.Notifications[i].RegistrationOptions)
		}
	}

	syntheticStructures := []Structure{}
	for i, regOptType := range registrationOptionTypes {
		if regOptType == nil || regOptType.Kind != kindAnd {
			continue
		}

		var ownerMethod string
		var ownerTypeName string
		for idx := range model.Requests {
			if model.Requests[idx].RegistrationOptions == regOptType {
				ownerMethod = model.Requests[idx].Method
				ownerTypeName = model.Requests[idx].TypeName
				break
			}
		}
		if ownerMethod == "" {
			for idx := range model.Notifications {
				if model.Notifications[idx].RegistrationOptions == regOptType {
					ownerMethod = model.Notifications[idx].Method
					ownerTypeName = model.Notifications[idx].TypeName
					break
				}
			}
		}
		if ownerMethod == "" {
			return errors.New("could not find owner for 'and' type registration option")
		}

		var structureName string
		if ownerTypeName != "" {
			structureName = strings.TrimSuffix(strings.TrimSuffix(ownerTypeName, "Request"), "Notification") + "RegistrationOptions"
		} else {
			parts := strings.Split(ownerMethod, "/")
			last := parts[len(parts)-1]
			structureName = goPublicIdentifier(last) + "RegistrationOptions"
		}

		refTypes := []*Type{}
		for _, item := range regOptType.Items {
			if item.Kind == kindReference {
				refTypes = append(refTypes, item)
			}
		}

		syntheticStructures = append(syntheticStructures, Structure{
			Name:          structureName,
			Properties:    nil,
			Extends:       refTypes,
			Documentation: fmt.Sprintf("Registration options for %s.", ownerMethod),
		})

		registrationOptionTypes[i] = &Type{
			Kind: kindReference,
			Name: structureName,
		}
	}

	neededDataStructures := NewOrderedMap[struct{}]()
	for si := range model.Structures {
		structure := &model.Structures[si]
		for pi := range structure.Properties {
			prop := &structure.Properties[pi]

			if prop.Name == "initializationOptions" && prop.Type != nil && prop.Type.Kind == kindReference && prop.Type.Name == "LSPAny" {
				prop.Type = &Type{
					Kind: kindReference,
					Name: "InitializationOptions",
				}
			}

			if prop.Name == "data" && prop.Type != nil && prop.Type.Kind == kindReference && prop.Type.Name == "LSPAny" {
				customDataType := structure.Name + "Data"
				prop.Type = &Type{
					Kind: kindReference,
					Name: customDataType,
				}
				if _, ok := explicitDataStructures()[customDataType]; !ok {
					neededDataStructures.Set(customDataType, struct{}{})
				}
			}

			if prop.Name == "registerOptions" && prop.Type != nil && prop.Type.Kind == kindReference && prop.Type.Name == "LSPAny" {
				if len(registrationOptionTypes) > 0 {
					registerOptionsUnionType = &Type{
						Kind:  kindOr,
						Items: registrationOptionTypes,
					}
					prop.Type = registerOptionsUnionType
				}
			}
		}
	}

	for _, dataTypeName := range neededDataStructures.Keys() {
		baseName := strings.TrimSuffix(dataTypeName, "Data")
		customStructures = append(customStructures, Structure{
			Name:       dataTypeName,
			Properties: nil,
			Documentation: fmt.Sprintf("%s is a placeholder for custom data preserved on a %s.",
				dataTypeName, baseName),
		})
	}

	model.Structures = append(model.Structures, customStructures...)
	model.Structures = append(model.Structures, syntheticStructures...)

	structureMap := make(map[string]*Structure, len(model.Structures))
	for i := range model.Structures {
		structureMap[model.Structures[i].Name] = &model.Structures[i]
	}

	var collectInheritedProperties func(structure *Structure, visited map[string]struct{}) []Property
	collectInheritedProperties = func(structure *Structure, visited map[string]struct{}) []Property {
		if structure == nil {
			return nil
		}
		if visited == nil {
			visited = make(map[string]struct{})
		}
		if _, ok := visited[structure.Name]; ok {
			return nil
		}
		visited[structure.Name] = struct{}{}

		var properties []Property
		inheritance := append([]*Type{}, structure.Extends...)
		inheritance = append(inheritance, structure.Mixins...)

		for _, t := range inheritance {
			if t == nil || t.Kind != kindReference {
				continue
			}
			inheritedStructure := structureMap[t.Name]
			if inheritedStructure != nil {
				nextVisited := make(map[string]struct{}, len(visited))
				maps.Copy(nextVisited, visited)
				properties = append(properties, collectInheritedProperties(inheritedStructure, nextVisited)...)
				properties = append(properties, inheritedStructure.Properties...)
			}
		}

		return properties
	}

	for si := range model.Structures {
		structure := &model.Structures[si]
		inheritedProperties := collectInheritedProperties(structure, nil)

		propertyMap := NewOrderedMap[Property]()
		for _, prop := range inheritedProperties {
			propertyMap.Set(prop.Name, prop)
		}
		for _, prop := range structure.Properties {
			propertyMap.Set(prop.Name, prop)
		}

		merged := make([]Property, 0, len(propertyMap.keys))
		for _, name := range propertyMap.keys {
			merged = append(merged, propertyMap.values[name])
		}
		structure.Properties = merged
		structure.Extends = nil
		structure.Mixins = nil

		if structure.Name == "ServerCapabilities" || structure.Name == "ClientCapabilities" {
			filtered := structure.Properties[:0]
			for _, prop := range structure.Properties {
				if prop.Name == "experimental" {
					continue
				}
				filtered = append(filtered, prop)
			}
			structure.Properties = filtered
		}
	}

	filtered := model.Structures[:0]
	for _, s := range model.Structures {
		if s.Name == "_InitializeParams" {
			continue
		}
		filtered = append(filtered, s)
	}
	model.Structures = filtered

	return nil
}

type GoType struct {
	Name         string
	NeedsPointer bool
}

type UnionMember struct {
	Name          string
	Type          *Type
	ContainedNull bool
}

type TypeInfo struct {
	Types        map[string]GoType
	LiteralTypes *OrderedMap[string]
	UnionTypes   *OrderedMap[[]UnionMember]
	TypeAliasMap map[string]*Type
}

func newTypeInfo() TypeInfo {
	return TypeInfo{
		Types:        map[string]GoType{},
		LiteralTypes: NewOrderedMap[string](),
		UnionTypes:   NewOrderedMap[[]UnionMember](),
		TypeAliasMap: map[string]*Type{},
	}
}

var typeInfo TypeInfo

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// commonInitialisms is based on the set used by golint and the Google Go Style Guide.
// It is used to produce idiomatic Go identifiers from schema / JSON names.
var commonInitialisms = map[string]struct{}{
	"API":   {},
	"ASCII": {},
	"CPU":   {},
	"CSS":   {},
	"DNS":   {},
	"EOF":   {},
	"GUID":  {},
	"HTML":  {},
	"HTTP":  {},
	"HTTPS": {},
	"ID":    {},
	"IP":    {},
	"JSON":  {},
	"LHS":   {},
	"QPS":   {},
	"RAM":   {},
	"RHS":   {},
	"RPC":   {},
	"SLA":   {},
	"SMTP":  {},
	"SQL":   {},
	"SSH":   {},
	"TCP":   {},
	"TLS":   {},
	"TTL":   {},
	"UDP":   {},
	"UI":    {},
	"UID":   {},
	"UUID":  {},
	"URI":   {},
	"URL":   {},
	"UTF8":  {},
	"VM":    {},
	"XML":   {},
	"XMPP":  {},
	"XSRF":  {},
	"XSS":   {},
	"LSP":   {},
}

func goPublicIdentifier(name string) string {
	if name == "" {
		return ""
	}

	words := splitIdentifierWords(name)
	if len(words) == 0 {
		return ""
	}

	var b strings.Builder
	for _, word := range words {
		if word == "" {
			continue
		}

		upper := strings.ToUpper(word)
		if _, ok := commonInitialisms[upper]; ok {
			b.WriteString(upper)
			continue
		}

		// Handle plural initialisms: "Ids" -> "IDs", "Uris" -> "URIs".
		if strings.HasSuffix(word, "s") && len(word) > 1 {
			base := word[:len(word)-1]
			baseUpper := strings.ToUpper(base)
			if _, ok := commonInitialisms[baseUpper]; ok {
				b.WriteString(baseUpper)
				b.WriteByte('s')
				continue
			}
			// Preserve pluralized acronyms not in the initialism list: "ABAPs" -> "ABAPs".
			if baseUpper == base && len(base) > 1 {
				b.WriteString(base)
				b.WriteByte('s')
				continue
			}
		}

		// Preserve already-uppercase words (e.g. ABAP, CPP) that aren't in the
		// initialism list.
		if upper == word {
			b.WriteString(word)
			continue
		}

		lower := strings.ToLower(word)
		b.WriteString(strings.ToUpper(lower[:1]))
		b.WriteString(lower[1:])
	}

	out := b.String()
	if out == "" {
		return ""
	}

	// Avoid invalid identifiers if we ever encounter a name starting with a digit.
	if out[0] >= '0' && out[0] <= '9' {
		return "X" + out
	}
	return out
}

func splitIdentifierWords(s string) []string {
	if s == "" {
		return nil
	}

	var words []string
	start := 0

	flush := func(i int) {
		if start < i {
			words = append(words, s[start:i])
		}
	}

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if !isAlphaNum(ch) {
			flush(i)
			start = i + 1
			continue
		}
		if i == start {
			continue
		}

		prev := s[i-1]
		if !isAlphaNum(prev) {
			continue
		}

		if isUpper(ch) && isDigit(prev) {
			flush(i)
			start = i
			continue
		}
		if isUpper(ch) && isLower(prev) {
			flush(i)
			start = i
			continue
		}
		if isUpper(ch) && isUpper(prev) && i+1 < len(s) && isLower(s[i+1]) {
			// Split "JSONText" into ["JSON", "Text"].
			flush(i)
			start = i
			continue
		}
	}
	flush(len(s))
	return words
}

func isUpper(b byte) bool { return b >= 'A' && b <= 'Z' }

func isLower(b byte) bool { return b >= 'a' && b <= 'z' }

func isLetter(b byte) bool { return isUpper(b) || isLower(b) }

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func isAlphaNum(b byte) bool { return isLetter(b) || isDigit(b) }

func resolveType(t *Type) GoType {
	switch t.Kind {
	case kindBase:
		switch t.Name {
		case "integer":
			return GoType{
				Name:         "int32",
				NeedsPointer: false,
			}

		case "uinteger":
			return GoType{
				Name:         "uint32",
				NeedsPointer: false,
			}

		case "string":
			return GoType{
				Name:         "string",
				NeedsPointer: false,
			}

		case "boolean":
			return GoType{
				Name:         "bool",
				NeedsPointer: false,
			}

		case "URI":
			return GoType{
				Name:         "URI",
				NeedsPointer: false,
			}

		case "DocumentUri":
			return GoType{
				Name:         "DocumentURI",
				NeedsPointer: false,
			}

		case "decimal":
			return GoType{
				Name:         "float64",
				NeedsPointer: false,
			}

		case "null":
			return GoType{
				Name:         "any",
				NeedsPointer: false,
			}

		default:
			panic(fmt.Sprintf("Unsupported base type: %s", t.Name))
		}

	case kindReference:
		if override, ok := typeAliasOverrides[t.Name]; ok {
			return override
		}
		if aliased, ok := typeInfo.TypeAliasMap[t.Name]; ok {
			return resolveType(aliased)
		}
		if refType, ok := typeInfo.Types[t.Name]; ok {
			return refType
		}
		refType := GoType{Name: goPublicIdentifier(t.Name), NeedsPointer: true}
		typeInfo.Types[t.Name] = refType
		return refType

	case kindArray:
		elem := resolveType(t.Element)
		name := "[]" + elem.Name
		if elem.NeedsPointer {
			name = "[]*" + elem.Name
		}
		return GoType{
			Name:         name,
			NeedsPointer: false,
		}

	case kindMap:
		key := resolveType(t.Key)
		val := resolveType(t.Value)
		valName := val.Name
		if val.NeedsPointer {
			valName = "*" + valName
		}
		return GoType{
			Name:         fmt.Sprintf("map[%s]%s", key.Name, valName),
			NeedsPointer: false,
		}

	case kindTuple:
		if len(t.Items) == 2 && t.Items[0].Kind == kindBase && t.Items[0].Name == "uinteger" && t.Items[1].Kind == kindBase && t.Items[1].Name == "uinteger" {
			return GoType{
				Name:         "[2]uint32",
				NeedsPointer: false,
			}
		}
		panic(fmt.Sprintf("Unsupported tuple type: %v", t))

	case kindStringLiteral:
		typeName := "StringLiteral" + titleCase(*t.StringValue)
		typeInfo.LiteralTypes.Set(fmt.Sprintf("%v", *t.StringValue), typeName)
		return GoType{
			Name:         typeName,
			NeedsPointer: false,
		}

	case kindIntegerLiteral:
		typeName := fmt.Sprintf("IntegerLiteral%d", *t.IntegerValue)
		typeInfo.LiteralTypes.Set(fmt.Sprintf("%v", *t.IntegerValue), typeName)
		return GoType{
			Name:         typeName,
			NeedsPointer: false,
		}

	case kindBooleanLiteral:
		typeName := "BooleanLiteral"
		if *t.BoolValue {
			typeName += "True"
		} else {
			typeName += "False"
		}
		typeInfo.LiteralTypes.Set(fmt.Sprintf("%v", *t.BoolValue), typeName)
		return GoType{
			Name:         typeName,
			NeedsPointer: false,
		}

	case kindLiteral:
		if t.Literal != nil && len(t.Literal.Properties) == 0 {
			return GoType{"struct{}", false}
		}
		panic(fmt.Sprintf("Unexpected non-empty literal object: %v", t.Literal))

	case kindOr:
		return handleOrType(t)

	default:
		panic(fmt.Sprintf("Unsupported type kind: %s", t.Kind))
	}
}

func flattenOrTypes(types []*Type) []*Type {
	flattened := []*Type{}
	seen := make(map[*Type]struct{})

	add := func(raw *Type) {
		if raw == nil {
			return
		}

		t := raw
		if raw.Kind == kindReference {
			if aliased, ok := typeInfo.TypeAliasMap[raw.Name]; ok && aliased.Kind == kindOr {
				t = aliased
			}
		}

		if t.Kind == kindOr {
			for _, sub := range flattenOrTypes(t.Items) {
				if _, ok := seen[sub]; !ok {
					seen[sub] = struct{}{}
					flattened = append(flattened, sub)
				}
			}
			return
		}

		if _, ok := seen[raw]; !ok {
			seen[raw] = struct{}{}
			flattened = append(flattened, raw)
		}
	}

	for _, t := range types {
		add(t)
	}

	return flattened
}

func pluralize(name string) string {
	if strings.HasSuffix(name, "s") || strings.HasSuffix(name, "x") || strings.HasSuffix(name, "z") || strings.HasSuffix(name, "ch") || strings.HasSuffix(name, "sh") {
		return name + "es"
	}

	if strings.HasSuffix(name, "y") && len(name) > 1 {
		penultimate := name[len(name)-2 : len(name)-1]
		if !strings.ContainsAny(penultimate, "aeiou") {
			return name[:len(name)-1] + "ies"
		}
	}

	return name + "s"
}

func handleOrType(orType *Type) GoType {
	types := flattenOrTypes(orType.Items)

	nullIndex := -1
	for i, t := range types {
		if t.Kind == kindBase && t.Name == "null" {
			nullIndex = i
			break
		}
	}
	containedNull := nullIndex != -1

	nonNullTypes := types
	if containedNull {
		nonNullTypes = append([]*Type{}, types[:nullIndex]...)
		nonNullTypes = append(nonNullTypes, types[nullIndex+1:]...)
	}
	if len(nonNullTypes) == 0 {
		panic(fmt.Sprintf("Union type with only null is not supported: %v", types))
	}

	memberNames := make([]string, len(nonNullTypes))
	for i, t := range nonNullTypes {
		switch t.Kind {
		case kindReference:
			memberNames[i] = goPublicIdentifier(t.Name)

		case kindBase:
			memberNames[i] = goPublicIdentifier(t.Name)

		case kindArray:
			if t.Element.Kind == kindReference || t.Element.Kind == kindBase {
				memberNames[i] = pluralize(goPublicIdentifier(t.Element.Name))
			} else {
				elem := resolveType(t.Element)
				memberNames[i] = elem.Name + "Array"
			}

		case kindLiteral:
			if t.Literal != nil && len(t.Literal.Properties) == 0 {
				memberNames[i] = "EmptyObject"
			} else {
				panic(fmt.Sprintf("Unsupported type kind in union: %s", t.Kind))
			}

		case kindTuple:
			memberNames[i] = "Tuple"

		default:
			panic(fmt.Sprintf("Unsupported type kind in union: %s", t.Kind))
		}
	}

	findLongestCommonPrefix := func(names []string) string {
		if len(names) <= 1 {
			return ""
		}

		splitPascalCase := func(name string) []string {
			var chunks []string
			current := ""
			for i := 0; i < len(name); i++ {
				ch := name[i]
				if ch >= 'A' && ch <= 'Z' && current != "" {
					chunks = append(chunks, current)
					current = string(ch)
				} else {
					current += string(ch)
				}
			}
			if current != "" {
				chunks = append(chunks, current)
			}
			return chunks
		}

		allChunks := make([][]string, len(names))
		minLen := -1
		for i, n := range names {
			chunks := splitPascalCase(n)
			allChunks[i] = chunks
			if minLen == -1 || len(chunks) < minLen {
				minLen = len(chunks)
			}
		}

		var common []string
		for i := 0; i < minLen; i++ {
			chunk := allChunks[0][i]
			allMatch := true
			for _, c := range allChunks[1:] {
				if c[i] != chunk {
					allMatch = false
					break
				}
			}
			if allMatch {
				common = append(common, chunk)
			} else {
				break
			}
		}

		return strings.Join(common, "")
	}

	commonPrefix := findLongestCommonPrefix(memberNames)
	unionTypeName := ""

	if commonPrefix != "" {
		trimmed := make([]string, len(memberNames))
		allNonEmpty := true
		for i, name := range memberNames {
			trimmed[i] = strings.TrimPrefix(name, commonPrefix)
			if trimmed[i] == "" {
				allNonEmpty = false
			}
		}
		if allNonEmpty {
			unionTypeName = commonPrefix + strings.Join(trimmed, "Or")
			memberNames = trimmed
		} else {
			unionTypeName = strings.Join(memberNames, "Or")
		}
	} else {
		unionTypeName = strings.Join(memberNames, "Or")
	}

	if orType == registerOptionsUnionType {
		unionTypeName = "RegisterOptions"
		for i, name := range memberNames {
			if before, ok := strings.CutSuffix(name, "RegistrationOptions"); ok {
				memberNames[i] = before
			}
		}
	}

	if containedNull {
		unionTypeName += "OrNull"
	} else {
		containedNull = false
	}

	union := make([]UnionMember, len(memberNames))
	for i, name := range memberNames {
		union[i] = UnionMember{
			Name:          name,
			Type:          nonNullTypes[i],
			ContainedNull: containedNull,
		}
	}

	typeInfo.UnionTypes.Set(unionTypeName, union)

	return GoType{
		Name:         unionTypeName,
		NeedsPointer: false,
	}
}

var typeAliasOverrides = map[string]GoType{
	"LSPAny": {
		Name:         "any",
		NeedsPointer: false,
	},
	"LSPArray": {
		Name:         "[]any",
		NeedsPointer: false,
	},
	"LSPObject": {
		Name:         "map[string]any",
		NeedsPointer: false,
	},
	"uint64": {
		Name:         "uint64",
		NeedsPointer: false,
	},
}

func collectTypeDefinitions(model *MetaModel) {
	for _, enumeration := range model.Enumerations {
		typeInfo.Types[enumeration.Name] = GoType{
			Name:         goPublicIdentifier(enumeration.Name),
			NeedsPointer: false,
		}
	}

	valueTypes := map[string]struct{}{
		"Position":                            {},
		"Range":                               {},
		"Location":                            {},
		"Color":                               {},
		"TextDocumentIdentifier":              {},
		"NotebookDocumentIdentifier":          {},
		"PreviousResultId":                    {},
		"VersionedNotebookDocumentIdentifier": {},
		"VersionedTextDocumentIdentifier":     {},
		"OptionalVersionedTextDocumentIdentifier": {},
		"ExportInfoMapKey":                        {},
	}

	for _, structure := range model.Structures {
		_, isValue := valueTypes[structure.Name]
		typeInfo.Types[structure.Name] = GoType{
			Name:         goPublicIdentifier(structure.Name),
			NeedsPointer: !isValue,
		}
	}

	for _, alias := range model.TypeAliases {
		if _, ok := typeAliasOverrides[alias.Name]; ok {
			continue
		}
		typeInfo.TypeAliasMap[alias.Name] = alias.Type
	}
}

var (
	spaceRE = regexp.MustCompile(`(\w ) +`)
	linkRE  = regexp.MustCompile(`\{@link(?:code)?.*?([^} ]+)\}`)
	tagRE   = regexp.MustCompile(`^@(since|proposed|deprecated)(.*)`)
)

func formatDocumentation(s string) string {
	if s == "" {
		return ""
	}

	var lines []string
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimRight(line, " \t\r")
		line = spaceRE.ReplaceAllString(line, "$1")
		line = linkRE.ReplaceAllString(line, "$1")
		if matches := tagRE.FindStringSubmatch(line); len(matches) > 0 {
			lines = append(lines, "")
			tag := titleCase(matches[1])
			rest := matches[2]
			if rest != "" {
				line = fmt.Sprintf("%s:%s", tag, rest)
			} else {
				line = fmt.Sprintf("%s.", tag)
			}
		}
		lines = append(lines, line)
	}

	for {
		removed := false
		for i := 0; i < len(lines); i++ {
			if lines[i] != "" {
				continue
			}
			if i == 0 || i == len(lines)-1 || !(lines[i-1] != "" && lines[i+1] != "") {
				lines = append(lines[:i], lines[i+1:]...)
				removed = true
				break
			}
		}
		if !removed {
			break
		}
	}

	if len(lines) == 0 {
		return ""
	}

	return "// " + strings.Join(lines, "\n// ") + "\n"
}

func methodNameIdentifier(name string) string {
	parts := strings.Split(name, "/")
	for i, p := range parts {
		if p == "$" {
			parts[i] = ""
		} else {
			parts[i] = goPublicIdentifier(p)
		}
	}
	return strings.Join(parts, "")
}

type codeWriter struct {
	b strings.Builder
}

const generatedHeader = `// Copyright 2025 The mcp-lsp Authors.
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

// Code generated by lspgen.go; DO NOT EDIT.

`

func (w *codeWriter) Write(s string) {
	w.b.WriteString(s)
}

func (w *codeWriter) WriteLine(s string) {
	w.b.WriteString(s)
	w.b.WriteByte('\n')
}

func generateCode(model *MetaModel) string {
	w := &codeWriter{}

	write := w.Write
	writeLine := w.WriteLine

	generateResolvedStruct := func(structure *Structure, indent string) []string {
		var lines []string
		for _, prop := range structure.Properties {
			if prop.Documentation != "" {
				propDoc := formatDocumentation(prop.Documentation)
				if propDoc != "" {
					for line := range strings.SplitSeq(strings.TrimRight(propDoc, "\n"), "\n") {
						lines = append(lines, indent+line)
					}
				}
			}

			fieldName := goPublicIdentifier(prop.Name)

			t := resolveType(prop.Type)
			if prop.Type.Kind == kindReference {
				// Use the model (TypeScript) name for lookups, but the Go name for emitted identifiers.
				if refStructure := findStructure(model.Structures, prop.Type.Name); refStructure != nil {
					lines = append(lines, fmt.Sprintf("%s%s Resolved%s `json:\"%s,omitzero\"`", indent, fieldName, t.Name, prop.Name))
					continue
				}
			}
			lines = append(lines, fmt.Sprintf("%s%s %s `json:\"%s,omitzero\"`", indent, fieldName, t.Name, prop.Name))
		}

		return lines
	}

	generateResolveConversion := func(structure *Structure, varName, indent string) []string {
		var lines []string
		for _, prop := range structure.Properties {
			t := resolveType(prop.Type)
			fieldName := goPublicIdentifier(prop.Name)
			access := varName + "." + fieldName
			if prop.Type.Kind == kindReference {
				if refStructure := findStructure(model.Structures, prop.Type.Name); refStructure != nil {
					lines = append(lines, fmt.Sprintf("%s%s: resolve%s(%s),", indent, fieldName, t.Name, access))
					continue
				}
			}

			if prop.Optional || t.NeedsPointer {
				lines = append(lines, fmt.Sprintf("%s%s: derefOr(%s),", indent, fieldName, access))
			} else {
				lines = append(lines, fmt.Sprintf("%s%s: %s,", indent, fieldName, access))
			}
		}

		return lines
	}

	var collectStructureDependencies func(structure *Structure, visited map[string]struct{}) []*Structure
	collectStructureDependencies = func(structure *Structure, visited map[string]struct{}) []*Structure {
		if structure == nil {
			return nil
		}
		if visited == nil {
			visited = make(map[string]struct{})
		}
		if _, ok := visited[structure.Name]; ok {
			return nil
		}
		visited[structure.Name] = struct{}{}

		var deps []*Structure
		for _, prop := range structure.Properties {
			if prop.Type.Kind == kindReference {
				ref := findStructure(model.Structures, prop.Type.Name)
				if ref != nil {
					nextVisited := make(map[string]struct{}, len(visited))
					maps.Copy(nextVisited, visited)
					deps = append(deps, collectStructureDependencies(ref, nextVisited)...)
					deps = append(deps, ref)
				}
			}
		}
		return deps
	}

	generateResolvedTypeAndHelper := func(structure *Structure, isMain bool) []string {
		var lines []string
		goStructureName := goPublicIdentifier(structure.Name)
		typeName := "Resolved" + goStructureName
		funcName := "resolve" + goStructureName
		if isMain {
			funcName = "Resolve" + goStructureName
		}

		if !isMain {
			if structure.Documentation != "" {
				typeDoc := formatDocumentation(structure.Documentation)
				lines = append(lines, fmt.Sprintf("// %s is a resolved version of %s with all optional fields", typeName, goStructureName))
				lines = append(lines, "// converted to non-pointer values for easier access.")
				lines = append(lines, "//")
				if typeDoc != "" {
					lines = append(lines, strings.TrimRight(typeDoc, "\n"))
				}
			} else {
				lines = append(lines, fmt.Sprintf("// %s is a resolved version of %s with all optional fields", typeName, goStructureName))
				lines = append(lines, "// converted to non-pointer values for easier access.")
			}
		}

		lines = append(lines, fmt.Sprintf("type %s struct {", typeName))
		lines = append(lines, generateResolvedStruct(structure, "\t")...)
		lines = append(lines, "}")
		lines = append(lines, "")

		lines = append(lines, fmt.Sprintf("func %s(v *%s) %s {", funcName, goStructureName, typeName))
		lines = append(lines, "\tif v == nil {")
		lines = append(lines, fmt.Sprintf("\t\treturn %s{}", typeName))
		lines = append(lines, "\t}")
		lines = append(lines, fmt.Sprintf("\treturn %s{", typeName))
		lines = append(lines, generateResolveConversion(structure, "v", "\t\t")...)
		lines = append(lines, "\t}")
		lines = append(lines, "}")
		lines = append(lines, "")
		return lines
	}

	write(generatedHeader)
	writeLine("package protocol")
	writeLine("")
	writeLine("import (")
	writeLine("\t\"fmt\"")
	writeLine("\t\"strings\"")
	writeLine("")
	writeLine("\t\"github.com/go-json-experiment/json\"")
	writeLine("\t\"github.com/go-json-experiment/json/jsontext\"")
	writeLine(")")
	writeLine("")
	writeLine("// Meta model version " + model.MetaData.Version)
	writeLine("")
	writeLine("// Structures")
	writeLine("")

	for si := range model.Structures {
		structure := &model.Structures[si]
		goStructureName := goPublicIdentifier(structure.Name)
		generateStructFields := func(name string, includeDocumentation bool) {
			if includeDocumentation {
				write(formatDocumentation(structure.Documentation))
			}
			writeLine("type " + name + " struct {")
			for _, prop := range structure.Properties {
				if includeDocumentation {
					write(formatDocumentation(prop.Documentation))
				}
				t := resolveType(prop.Type)
				useOmitzero := prop.Optional || prop.OmitzeroValue
				goType := t.Name
				if (prop.Optional || t.NeedsPointer) && !prop.OmitzeroValue {
					goType = "*" + goType
				}
				tag := prop.Name
				if useOmitzero {
					tag += ",omitzero"
				}
				writeLine(fmt.Sprintf("\t%s %s `json:\"%s\"`", goPublicIdentifier(prop.Name), goType, tag))
				if includeDocumentation {
					writeLine("")
				}
			}
			writeLine("}")
			writeLine("")
		}

		generateStructFields(goStructureName, true)
		writeLine("")

		if hasTextDocumentURI(structure) {
			writeLine(fmt.Sprintf("func (s *%s) TextDocumentURI() DocumentURI {", goStructureName))
			writeLine("\treturn s.TextDocument.URI")
			writeLine("}")
			writeLine("")
			if hasTextDocumentPosition(structure) {
				writeLine(fmt.Sprintf("func (s *%s) TextDocumentPosition() Position {", goStructureName))
				writeLine("\treturn s.Position")
				writeLine("}")
				writeLine("")
			}
		}

		requiredProps := []Property{}
		for _, p := range structure.Properties {
			if p.Optional || p.OmitzeroValue {
				continue
			}
			requiredProps = append(requiredProps, p)
		}

		if len(requiredProps) > 0 {
			writeLine(fmt.Sprintf("\tvar _ json.UnmarshalerFrom = (*%s)(nil)", goStructureName))
			writeLine("")
			writeLine(fmt.Sprintf("func (s *%s) UnmarshalJSONFrom(dec *jsontext.Decoder) error {", goStructureName))
			writeLine("\tconst (")
			for i, prop := range requiredProps {
				iotaPrefix := ""
				if i == 0 {
					iotaPrefix = " uint = 1 << iota"
				}
				writeLine(fmt.Sprintf("\t\tmissing%s%s", goPublicIdentifier(prop.Name), iotaPrefix))
			}
			writeLine("\t\t_missingLast")
			writeLine("\t)")
			writeLine("\tmissing := _missingLast - 1")
			writeLine("")
			writeLine("\tif k := dec.PeekKind(); k != '{' {")
			writeLine("\t\treturn fmt.Errorf(\"expected object start, but encountered %v\", k)")
			writeLine("\t}")
			writeLine("\tif _, err := dec.ReadToken(); err != nil {")
			writeLine("\t\treturn err")
			writeLine("\t}")
			writeLine("")
			writeLine("\tfor dec.PeekKind() != '}' {")
			writeLine("\tname, err := dec.ReadValue()")
			writeLine("\t\tif err != nil {")
			writeLine("\t\t\treturn err")
			writeLine("\t\t}")
			writeLine("\t\tswitch string(name) {")
			for _, prop := range structure.Properties {
				writeLine(fmt.Sprintf("\t\tcase `\"%s\"`:", prop.Name))
				if !prop.Optional && !prop.OmitzeroValue {
					writeLine(fmt.Sprintf("\t\t\tmissing &^= missing%s", goPublicIdentifier(prop.Name)))
				}
				writeLine(fmt.Sprintf("\t\t\tif err := json.UnmarshalDecode(dec, &s.%s); err != nil {", goPublicIdentifier(prop.Name)))
				writeLine("\t\t\t\treturn err")
				writeLine("\t\t\t}")
			}
			writeLine("\t\tdefault:")
			writeLine("\t\t// Ignore unknown properties.")
			writeLine("\t\t}")
			writeLine("\t}")
			writeLine("")
			writeLine("\tif _, err := dec.ReadToken(); err != nil {")
			writeLine("\t\treturn err")
			writeLine("\t}")
			writeLine("")
			writeLine("\tif missing != 0 {")
			writeLine("\t\tvar missingProps []string")
			for _, prop := range requiredProps {
				writeLine(fmt.Sprintf("\t\tif missing&missing%s != 0 {", goPublicIdentifier(prop.Name)))
				writeLine(fmt.Sprintf("\t\t\tmissingProps = append(missingProps, \"%s\")", prop.Name))
				writeLine("\t\t}")
			}
			writeLine("\t\treturn fmt.Errorf(\"missing required properties: %s\", strings.Join(missingProps, \", \") )")
			writeLine("\t}")
			writeLine("")
			writeLine("\treturn nil")
			writeLine("}")
			writeLine("")
		}
	}

	bitflagEnums := map[string]struct{}{
		"WatchKind": {},
	}
	isBitflagEnum := func(enum Enumeration) bool {
		_, ok := bitflagEnums[enum.Name]
		return ok
	}

	writeLine("// Enumerations")
	writeLine("")

	for _, enumeration := range model.Enumerations {
		write(formatDocumentation(enumeration.Documentation))
		goEnumName := goPublicIdentifier(enumeration.Name)
		var baseType string
		switch enumeration.Type.Name {
		case "string":
			baseType = "string"
		case "integer":
			baseType = "int32"
		case "uinteger":
			baseType = "uint32"
		default:
			panic(fmt.Sprintf("Unsupported enum type: %s", enumeration.Type.Name))
		}
		writeLine(fmt.Sprintf("type %s %s", goEnumName, baseType))
		writeLine("")

		enumValues := make([]NumericValue, len(enumeration.Values))
		for i, value := range enumeration.Values {
			asNumber, _ := toNumber(value.ValueRaw)
			goValueName := goPublicIdentifier(value.Name)
			enumValues[i] = NumericValue{
				Value:         fmt.Sprint(value.ValueRaw),
				NumericValue:  asNumber,
				Name:          goValueName,
				Identifier:    goEnumName + goValueName,
				Documentation: value.Documentation,
			}
		}

		writeLine("const (")
		for _, entry := range enumValues {
			write(formatDocumentation(entry.Documentation))
			valueLiteral := entry.Value
			if enumeration.Type.Name == "string" {
				valueLiteral = strings.TrimSuffix(strings.TrimPrefix(entry.Value, "\""), "\"")
				valueLiteral = fmt.Sprintf("\"%s\"", valueLiteral)
			}
			writeLine(fmt.Sprintf("\t%s %s = %s", entry.Identifier, goEnumName, valueLiteral))
		}
		writeLine(")")

		writeLine("")

		if enumeration.Type.Name != "string" {
			if isBitflagEnum(enumeration) {
				sorted := append([]NumericValue{}, enumValues...)
				sortByNumericValue(sorted)
				names := make([]string, len(sorted))
				values := make([]float64, len(sorted))
				for i, v := range sorted {
					names[i] = v.Name
					values[i] = v.NumericValue
				}
				nameConst := "_" + goEnumName + "_name"
				indexVar := "_" + goEnumName + "_index"
				combinedNames := strings.Join(names, "")
				writeLine(fmt.Sprintf("const %s = \"%s\"", nameConst, combinedNames))
				write("var " + indexVar + " = [...]uint16{0")
				offset := 0
				for _, name := range names {
					offset += len(name)
					write(fmt.Sprintf(", %d", offset))
				}
				writeLine("}")
				writeLine("")

				writeLine(fmt.Sprintf("func (e %s) String() string {", goEnumName))
				writeLine("\tif e == 0 {")
				writeLine("\t\treturn \"0\"")
				writeLine("\t}")
				writeLine("\tvar parts []string")
				for i, val := range values {
					writeLine(fmt.Sprintf("\tif e&%v != 0 {", val))
					writeLine(fmt.Sprintf("\t\tparts = append(parts, %s[%s[%d]:%s[%d+1]])", nameConst, indexVar, i, indexVar, i))
					writeLine("\t}")
				}
				writeLine("\tif len(parts) == 0 {")
				writeLine(fmt.Sprintf("\t\treturn fmt.Sprintf(\"%s(%%d)\", e)", goEnumName))
				writeLine("\t}")
				writeLine("\treturn strings.Join(parts, \"|\")")
				writeLine("}")
				writeLine("")
			} else {
				sortByNumericValue(enumValues)

				runs := splitNumeric(enumValues)
				nameConst := "_" + goEnumName + "_name"
				indexVar := "_" + goEnumName + "_index"

				if len(runs) == 1 {
					combined := strings.Builder{}
					for _, n := range runs[0].Names {
						combined.WriteString(n)
					}
					writeLine(fmt.Sprintf("const %s = \"%s\"", nameConst, combined.String()))
					write("var " + indexVar + " = [...]uint16{0")
					offset := 0
					for _, n := range runs[0].Names {
						offset += len(n)
						write(fmt.Sprintf(", %d", offset))
					}
					writeLine("}")
					writeLine("")
					minVal := runs[0].Values[0]
					writeLine(fmt.Sprintf("func (e %s) String() string {", goEnumName))
					writeLine(fmt.Sprintf("\ti := int(e) - %d", int(minVal)))
					writeLine(fmt.Sprintf("\tif i < 0 || i >= len(%s)-1 {", indexVar))
					writeLine(fmt.Sprintf("\t\treturn fmt.Sprintf(\"%s(%%d)\", e)", goEnumName))
					writeLine("\t}")
					writeLine(fmt.Sprintf("\treturn %s[%s[i]:%s[i+1]]", nameConst, indexVar, indexVar))
					writeLine("}")
					writeLine("")
				} else if len(runs) <= 10 {
					var allNames strings.Builder
					type runinfo struct {
						StartOffset int
						EndOffset   int
						MinVal      int
						MaxVal      int
					}
					runInfo := make([]runinfo, len(runs))
					offset := 0
					for i, run := range runs {
						runInfo[i].StartOffset = offset
						for _, n := range run.Names {
							offset += len(n)
							allNames.WriteString(n)
						}
						runInfo[i].EndOffset = offset
						runInfo[i].MinVal = int(run.Values[0])
						runInfo[i].MaxVal = int(run.Values[len(run.Values)-1])
					}
					writeLine(fmt.Sprintf("const %s = \"%s\"", nameConst, allNames.String()))
					writeLine("")
					for i, run := range runs {
						idx := fmt.Sprintf("%s_%d", indexVar, i)
						write("var " + idx + " = [...]uint16{0")
						offset := 0
						for _, n := range run.Names {
							offset += len(n)
							write(fmt.Sprintf(", %d", offset))
						}
						writeLine("}")
					}
					writeLine("")

					writeLine(fmt.Sprintf("func (e %s) String() string {", goEnumName))
					writeLine("\tswitch {")
					for i, run := range runs {
						info := runInfo[i]
						idx := fmt.Sprintf("%s_%d", indexVar, i)
						if len(run.Values) == 1 {
							writeLine(fmt.Sprintf("\tcase e == %d:", int(run.Values[0])))
							writeLine(fmt.Sprintf("\t\treturn %s[%d:%d]", nameConst, info.StartOffset, info.EndOffset))
						} else {
							if info.MinVal == 0 && strings.HasPrefix(baseType, "uint") {
								writeLine(fmt.Sprintf("\tcase e <= %d:", info.MaxVal))
							} else if info.MinVal == 0 {
								writeLine(fmt.Sprintf("\tcase 0 <= e && e <= %d:", info.MaxVal))
							} else {
								writeLine(fmt.Sprintf("\tcase %d <= e && e <= %d:", info.MinVal, info.MaxVal))
							}
							writeLine(fmt.Sprintf("\t\ti := int(e) - %d", info.MinVal))
							writeLine(fmt.Sprintf("\t\treturn %s[%d+%s[i]:%d+%s[i+1]]", nameConst, info.StartOffset, idx, info.StartOffset, idx))
						}
					}
					writeLine("\tdefault:")
					writeLine(fmt.Sprintf("\t\treturn fmt.Sprintf(\"%s(%%d)\", e)", goEnumName))
					writeLine("\t}")
					writeLine("}")
					writeLine("")
				} else {
					var allNames strings.Builder
					type valueMapEntry struct {
						Value       int
						StartOffset int
						EndOffset   int
					}
					var valueMap []valueMapEntry
					offset := 0
					for _, run := range runs {
						for i, name := range run.Names {
							start := offset
							offset += len(name)
							valueMap = append(valueMap, valueMapEntry{
								Value:       int(run.Values[i]),
								StartOffset: start,
								EndOffset:   offset,
							})
							allNames.WriteString(name)
						}
					}
					writeLine(fmt.Sprintf("const %s = \"%s\"", nameConst, allNames.String()))
					writeLine("")
					writeLine(fmt.Sprintf("var %s_map = map[%s]string{", goEnumName, goEnumName))
					for _, entry := range valueMap {
						writeLine(fmt.Sprintf("\t%d: %s[%d:%d],", entry.Value, nameConst, entry.StartOffset, entry.EndOffset))
					}
					writeLine("}")
					writeLine("")
					writeLine(fmt.Sprintf("func (e %s) String() string {", goEnumName))
					writeLine(fmt.Sprintf("\tif str, ok := %s_map[e]; ok {", goEnumName))
					writeLine("\t\treturn str")
					writeLine("\t}")
					writeLine(fmt.Sprintf("\treturn fmt.Sprintf(\"%s(%%d)\", e)", goEnumName))
					writeLine("}")
					writeLine("")
				}
			}
		}
	}

	requestsAndNotifications := append([]Request{}, model.Requests...)
	for _, n := range model.Notifications {
		requestsAndNotifications = append(requestsAndNotifications, Request{
			Method:              n.Method,
			TypeName:            n.TypeName,
			Params:              n.Params,
			ParamsArray:         n.ParamsArray,
			RegistrationMethod:  n.RegistrationMethod,
			RegistrationOptions: n.RegistrationOptions,
			MessageDirection:    n.MessageDirection,
			Documentation:       n.Documentation,
			Since:               n.Since,
			SinceTags:           n.SinceTags,
			Proposed:            n.Proposed,
			Deprecated:          n.Deprecated,
		})
	}

	writeLine("func unmarshalParams(method Method, data []byte) (any, error) {")
	writeLine("\tswitch method {")
	for _, request := range requestsAndNotifications {
		methodName := methodNameIdentifier(request.Method)
		if request.Params == nil && len(request.ParamsArray) == 0 {
			writeLine(fmt.Sprintf("\tcase Method%s:", methodName))
			writeLine("\t\treturn unmarshalEmpty(data)")
			continue
		}
		if len(request.ParamsArray) > 0 {
			panic(fmt.Sprintf("Unexpected array type for request params: %v", request.ParamsArray))
		}
		resolved := resolveType(request.Params)
		writeLine(fmt.Sprintf("\tcase Method%s:", methodName))
		if resolved.Name == "any" {
			writeLine("\t\treturn unmarshalAny(data)")
		} else {
			writeLine(fmt.Sprintf("\t\treturn unmarshalPtrTo[%s](data)", resolved.Name))
		}
	}
	writeLine("\tdefault:")
	writeLine("\t\treturn unmarshalAny(data)")
	writeLine("\t}")
	writeLine("}")
	writeLine("")

	writeLine("func unmarshalResult(method Method, data []byte) (any, error) {")
	writeLine("\tswitch method {")
	for _, request := range model.Requests {
		methodName := methodNameIdentifier(request.Method)
		responseTypeName := ""
		if request.TypeName != "" && strings.HasSuffix(request.TypeName, "Request") {
			responseTypeName = strings.TrimSuffix(request.TypeName, "Request") + "Response"
		} else {
			responseTypeName = methodName + "Response"
		}
		responseTypeName = goPublicIdentifier(responseTypeName)
		writeLine(fmt.Sprintf("\tcase Method%s:", methodName))
		writeLine(fmt.Sprintf("\t\treturn unmarshalValue[%s](data)", responseTypeName))
	}
	writeLine("\tdefault:")
	writeLine("\t\treturn unmarshalAny(data)")
	writeLine("\t}")
	writeLine("}")
	writeLine("")

	writeLine("// Methods")
	writeLine("const (")
	for _, request := range requestsAndNotifications {
		write(formatDocumentation(request.Documentation))
		methodName := methodNameIdentifier(request.Method)
		writeLine(fmt.Sprintf("\tMethod%s Method = \"%s\"", methodName, request.Method))
	}
	writeLine(")")
	writeLine("")

	writeLine("// Request response types")
	writeLine("")

	for _, request := range requestsAndNotifications {
		methodName := methodNameIdentifier(request.Method)
		var responseTypeName string
		hasResult := false
		for i := range model.Requests {
			if model.Requests[i].Method == request.Method {
				hasResult = true
				if model.Requests[i].TypeName != "" && strings.HasSuffix(model.Requests[i].TypeName, "Request") {
					responseTypeName = strings.TrimSuffix(model.Requests[i].TypeName, "Request") + "Response"
				} else {
					responseTypeName = methodName + "Response"
				}
				break
			}
		}
		if hasResult {
			responseTypeName = goPublicIdentifier(responseTypeName)
		}

		if hasResult {
			writeLine(fmt.Sprintf("// Response type for `%s`", request.Method))
			if request.Result != nil && request.Result.Kind == kindBase && request.Result.Name == "null" {
				writeLine(fmt.Sprintf("type %s = Null", responseTypeName))
			} else if request.Result != nil {
				resultType := resolveType(request.Result)
				goType := resultType.Name
				if resultType.NeedsPointer {
					goType = "*" + goType
				}
				writeLine(fmt.Sprintf("type %s = %s", responseTypeName, goType))
			}
			writeLine("")
		}

		if len(request.ParamsArray) > 0 {
			panic(fmt.Sprintf("Unexpected request params for %s: %v", methodName, request.ParamsArray))
		}
		paramType := request.Params
		paramGoType := "any"
		if paramType != nil {
			resolved := resolveType(paramType)
			paramGoType = resolved.Name
			if resolved.NeedsPointer {
				paramGoType = "*" + paramGoType
			}
		}
		writeLine(fmt.Sprintf("// Type mapping info for `%s`", request.Method))
		if hasResult {
			writeLine(fmt.Sprintf("var %sInfo = RequestInfo[%s, %s]{Method: Method%s}", methodName, paramGoType, responseTypeName, methodName))
		} else {
			writeLine(fmt.Sprintf("var %sInfo = NotificationInfo[%s]{Method: Method%s}", methodName, paramGoType, methodName))
		}
		writeLine("")
	}

	writeLine("// Union types")
	writeLine("")

	for i := 0; i < len(typeInfo.UnionTypes.keys); i++ {
		name := typeInfo.UnionTypes.keys[i]
		members, _ := typeInfo.UnionTypes.Get(name)
		writeLine(fmt.Sprintf("type %s struct {", name))
		uniqueTypeFields := NewOrderedMap[string]()
		for _, member := range members {
			t := resolveType(member.Type)
			memberType := t.Name
			if _, ok := uniqueTypeFields.Get(memberType); !ok {
				fieldName := titleCase(member.Name)
				uniqueTypeFields.Set(memberType, fieldName)
				writeLine(fmt.Sprintf("\t%s *%s", fieldName, memberType))
			}
		}
		writeLine("}")
		writeLine("")

		type fieldEntry struct {
			FieldName string
			TypeName  string
		}
		fieldEntries := make([]fieldEntry, 0, len(uniqueTypeFields.keys))
		for _, typeName := range uniqueTypeFields.keys {
			fieldEntries = append(fieldEntries, struct{ FieldName, TypeName string }{FieldName: uniqueTypeFields.values[typeName], TypeName: typeName})
		}

		writeLine(fmt.Sprintf("var _ json.MarshalerTo = (*%s)(nil)", name))
		writeLine("")
		writeLine(fmt.Sprintf("func (o *%s) MarshalJSONTo(enc *jsontext.Encoder) error {", name))
		unionContainedNull := false
		for _, member := range members {
			if member.ContainedNull {
				unionContainedNull = true
				break
			}
		}
		if unionContainedNull {
			write("\tassertAtMostOne(\"more than one element of " + name + " is set\", ")
		} else {
			write("\tassertOnlyOne(\"exactly one element of " + name + " should be set\", ")
		}
		for i, entry := range fieldEntries {
			if i > 0 {
				write(", ")
			}
			write("o." + entry.FieldName + " != nil")
		}
		writeLine(")")
		writeLine("")

		for _, entry := range fieldEntries {
			writeLine(fmt.Sprintf("\tif o.%s != nil {", entry.FieldName))
			writeLine(fmt.Sprintf("\t\treturn json.MarshalEncode(enc, o.%s)", entry.FieldName))
			writeLine("\t}")
		}
		if unionContainedNull {
			writeLine("\treturn enc.WriteToken(jsontext.Null)")
		} else {
			writeLine("\tpanic(\"unreachable\")")
		}
		writeLine("}")
		writeLine("")

		writeLine(fmt.Sprintf("var _ json.UnmarshalerFrom = (*%s)(nil)", name))
		writeLine("")
		writeLine(fmt.Sprintf("func (o *%s) UnmarshalJSONFrom(dec *jsontext.Decoder) error {", name))
		writeLine(fmt.Sprintf("\t*o = %s{}", name))
		writeLine("")
		writeLine("\tdata, err := dec.ReadValue()")
		writeLine("\tif err != nil {")
		writeLine("\t\treturn err")
		writeLine("\t}")
		if unionContainedNull {
			writeLine("\tif string(data) == \"null\" {")
			writeLine("\t\treturn nil")
			writeLine("\t}")
			writeLine("")
		}
		for _, entry := range fieldEntries {
			writeLine(fmt.Sprintf("\tvar v%s %s", entry.FieldName, entry.TypeName))
			writeLine(fmt.Sprintf("\tif err := json.Unmarshal(data, &v%s); err == nil {", entry.FieldName))
			writeLine(fmt.Sprintf("\t\to.%s = &v%s", entry.FieldName, entry.FieldName))
			writeLine("\t\treturn nil")
			writeLine("\t}")
		}
		writeLine(fmt.Sprintf("\treturn fmt.Errorf(\"invalid %s: %%s\", data)", name))
		writeLine("}")
		writeLine("")
	}

	writeLine("// Literal types")
	writeLine("")

	for i := 0; i < len(typeInfo.LiteralTypes.keys); i++ {
		value := typeInfo.LiteralTypes.keys[i]
		name, _ := typeInfo.LiteralTypes.Get(value)
		jsonValue, _ := json.Marshal(value)
		writeLine(fmt.Sprintf("// %s is a literal type for %s", name, string(jsonValue)))
		writeLine(fmt.Sprintf("type %s struct{}", name))
		writeLine("")
		writeLine(fmt.Sprintf("var _ json.MarshalerTo = %s{}", name))
		writeLine("")
		writeLine(fmt.Sprintf("func (o %s) MarshalJSONTo(enc *jsontext.Encoder) error {", name))
		writeLine(fmt.Sprintf("\treturn enc.WriteValue(jsontext.Value(`%s`))", string(jsonValue)))
		writeLine("}")
		writeLine("")
		writeLine(fmt.Sprintf("var _ json.UnmarshalerFrom = &%s{}", name))
		writeLine("")
		writeLine(fmt.Sprintf("func (o *%s) UnmarshalJSONFrom(dec *jsontext.Decoder) error {", name))
		writeLine("\tv, err := dec.ReadValue();")
		writeLine("\tif err != nil {")
		writeLine("\t\treturn err")
		writeLine("\t}")
		writeLine(fmt.Sprintf("\tif string(v) != `%s` {", string(jsonValue)))
		writeLine(fmt.Sprintf("\t\treturn fmt.Errorf(\"expected %s value %%s, got %%s\", `%s`, v)", name, string(jsonValue)))
		writeLine("\t}")
		writeLine("\treturn nil")
		writeLine("}")
		writeLine("")
	}

	clientCapsStructure := findStructure(model.Structures, "ClientCapabilities")
	if clientCapsStructure != nil {
		writeLine("// Helper function for dereferencing pointers with zero value fallback")
		writeLine("func derefOr[T any](v *T) T {")
		writeLine("\tif v != nil {")
		writeLine("\t\treturn *v")
		writeLine("\t}")
		writeLine("\tvar zero T")
		writeLine("\treturn zero")
		writeLine("}")
		writeLine("")

		deps := collectStructureDependencies(clientCapsStructure, nil)
		unique := NewOrderedMap[*Structure]()
		for _, dep := range deps {
			if dep != nil {
				unique.Set(dep.Name, dep)
			}
		}
		for _, name := range unique.keys {
			dep := unique.values[name]
			for _, line := range generateResolvedTypeAndHelper(dep, false) {
				writeLine(line)
			}
		}

		writeLine("// ResolvedClientCapabilities is a version of ClientCapabilities where all nested")
		writeLine("// fields are values (not pointers), making it easier to access deeply nested capabilities.")
		writeLine("// Use ResolveClientCapabilities to convert from ClientCapabilities.")
		if clientCapsStructure.Documentation != "" {
			writeLine("//")
			typeDoc := formatDocumentation(clientCapsStructure.Documentation)
			for line := range strings.SplitSeq(strings.TrimRight(typeDoc, "\n"), "\n") {
				writeLine(line)
			}
		}
		for _, line := range generateResolvedTypeAndHelper(clientCapsStructure, true) {
			writeLine(line)
		}
	}

	return w.b.String()
}

func findStructure(structures []Structure, name string) *Structure {
	for i := range structures {
		if structures[i].Name == name {
			return &structures[i]
		}
	}
	return nil
}

func hasSomeProp(structure *Structure, propName, propTypeName string) bool {
	for _, p := range structure.Properties {
		if p.Optional {
			continue
		}
		if p.Name == propName && p.Type.Kind == kindReference && p.Type.Name == propTypeName {
			return true
		}
	}
	return false
}

func hasTextDocumentURI(structure *Structure) bool {
	return hasSomeProp(structure, "textDocument", "TextDocumentIdentifier")
}

func hasTextDocumentPosition(structure *Structure) bool {
	return hasSomeProp(structure, "position", "Position")
}

func toNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case jsonv1.Number:
		f, err := n.Float64()
		if err == nil {
			return f, true
		}
	case string:
		var num float64
		if err := json.Unmarshal([]byte(n), &num); err == nil {
			return num, true
		}
	}
	return 0, false
}

type enumRun struct {
	Names  []string
	Values []float64
}

type NumericValue struct {
	Value         string
	NumericValue  float64
	Name          string
	Identifier    string
	Documentation string
}

func sortByNumericValue(values []NumericValue) {
	for i := range values {
		for j := i + 1; j < len(values); j++ {
			if values[j].NumericValue < values[i].NumericValue {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

func splitNumeric(values []NumericValue) []enumRun {
	if len(values) == 0 {
		return nil
	}

	runs := []enumRun{
		{
			Names:  []string{values[0].Name},
			Values: []float64{values[0].NumericValue},
		},
	}
	for i := 1; i < len(values); i++ {
		prev := values[i-1].NumericValue
		if values[i].NumericValue == prev+1 {
			runs[len(runs)-1].Names = append(runs[len(runs)-1].Names, values[i].Name)
			runs[len(runs)-1].Values = append(runs[len(runs)-1].Values, values[i].NumericValue)
		} else {
			runs = append(runs, enumRun{Names: []string{values[i].Name}, Values: []float64{values[i].NumericValue}})
		}
	}

	return runs
}

type outputMode string

const (
	outputModeWrite      outputMode = "write"
	outputModeStructList outputMode = "struct-list"
)

func structNamesFromSource(src []byte) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "generated.go", src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if _, ok := typeSpec.Type.(*ast.StructType); !ok {
				continue
			}
			names = append(names, typeSpec.Name.Name)
		}
	}

	return names, nil
}

func main() {
	modeFlag := flag.String("mode", string(outputModeWrite), "Output mode: write (default) or struct-list.")
	flag.Parse()

	mode := outputMode(*modeFlag)
	switch mode {
	case outputModeWrite, outputModeStructList:
	default:
		fmt.Fprintf(os.Stderr, "Invalid -mode %q (want %q or %q).\n", *modeFlag, outputModeWrite, outputModeStructList)
		os.Exit(2)
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get current filename")
	}
	baseDir := filepath.Dir(filename)
	out := filepath.Clean(filepath.Join(filepath.Dir(filepath.Dir(baseDir)), "lsp.go"))
	metaModelPath := filepath.Join(baseDir, "metaModel.json")

	metaModelData, err := os.ReadFile(metaModelPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Meta model file not found; did you forget to run fetchModel.mts?\n")
		os.Exit(1)
	}

	var model MetaModel
	if err := json.Unmarshal(metaModelData, &model); err != nil {
		panic(err)
	}

	customStructures = customStructures[:]
	registerOptionsUnionType = nil
	typeInfo = newTypeInfo()

	if err := patchAndPreprocessModel(&model); err != nil {
		panic(err)
	}
	collectTypeDefinitions(&model)
	generatedCode := generateCode(&model)

	data, err := gofumpt.Source([]byte(generatedCode), gofumpt.Options{
		LangVersion: "go1.25",
		ExtraRules:  true,
	})
	if err != nil {
		panic(err)
	}

	if mode == outputModeStructList {
		structs, err := structNamesFromSource(data)
		if err != nil {
			panic(err)
		}
		payload, err := jsonv1.Marshal(structs)
		if err != nil {
			panic(err)
		}
		fmt.Fprintln(os.Stdout, string(payload))
		return
	}

	if err := os.WriteFile(out, data, 0o644); err != nil {
		panic(err)
	}

	fmt.Printf("Successfully generated %s\n", out)
}
