// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

type metaModel struct {
	MetaData      metaData       `json:"metaData"`
	Requests      []request      `json:"requests"`
	Notifications []notification `json:"notifications"`
	Structures    []structure    `json:"structures"`
	Enumerations  []enumeration  `json:"enumerations"`
	TypeAliases   []typeAlias    `json:"typeAliases"`
}

type metaData struct {
	Version string `json:"version"`
}

type request struct {
	Method              string    `json:"method"`
	TypeName            string    `json:"typeName"`
	Result              *metaType `json:"result,omitempty"`
	Params              *metaType `json:"params,omitempty"`
	PartialResult       *metaType `json:"partialResult,omitempty"`
	RegistrationOptions *metaType `json:"registrationOptions,omitempty"`
	Documentation       string    `json:"documentation,omitempty"`
	MessageDirection    string    `json:"messageDirection,omitempty"`
	ClientCapability    string    `json:"clientCapability,omitempty"`
	ServerCapability    string    `json:"serverCapability,omitempty"`
	Since               string    `json:"since,omitempty"`
	Deprecated          string    `json:"deprecated,omitempty"`
	Proposed            bool      `json:"proposed,omitempty"`
}

type notification struct {
	Method           string    `json:"method"`
	TypeName         string    `json:"typeName"`
	Params           *metaType `json:"params,omitempty"`
	Documentation    string    `json:"documentation,omitempty"`
	MessageDirection string    `json:"messageDirection,omitempty"`
	ClientCapability string    `json:"clientCapability,omitempty"`
	ServerCapability string    `json:"serverCapability,omitempty"`
	Since            string    `json:"since,omitempty"`
	Deprecated       string    `json:"deprecated,omitempty"`
	Proposed         bool      `json:"proposed,omitempty"`
}

type structure struct {
	Name          string     `json:"name"`
	Properties    []property `json:"properties"`
	Extends       []metaType `json:"extends,omitempty"`
	Mixins        []metaType `json:"mixins,omitempty"`
	Documentation string     `json:"documentation,omitempty"`
	Since         string     `json:"since,omitempty"`
	Deprecated    string     `json:"deprecated,omitempty"`
	Proposed      bool       `json:"proposed,omitempty"`
}

type property struct {
	Name          string   `json:"name"`
	Type          metaType `json:"type"`
	Optional      bool     `json:"optional,omitempty"`
	Documentation string   `json:"documentation,omitempty"`
	Since         string   `json:"since,omitempty"`
	Deprecated    string   `json:"deprecated,omitempty"`
	Proposed      bool     `json:"proposed,omitempty"`
}

type enumeration struct {
	Name                 string             `json:"name"`
	Type                 metaType           `json:"type"`
	Values               []enumerationValue `json:"values"`
	SupportsCustomValues bool               `json:"supportsCustomValues,omitempty"`
	Documentation        string             `json:"documentation,omitempty"`
	Since                string             `json:"since,omitempty"`
	Deprecated           string             `json:"deprecated,omitempty"`
	Proposed             bool               `json:"proposed,omitempty"`
}

type enumerationValue struct {
	Name          string         `json:"name"`
	Value         jsontext.Value `json:"value"`
	Documentation string         `json:"documentation,omitempty"`
	Since         string         `json:"since,omitempty"`
	Deprecated    string         `json:"deprecated,omitempty"`
	Proposed      bool           `json:"proposed,omitempty"`
}

type typeAlias struct {
	Name          string   `json:"name"`
	Type          metaType `json:"type"`
	Documentation string   `json:"documentation,omitempty"`
	Since         string   `json:"since,omitempty"`
	Deprecated    string   `json:"deprecated,omitempty"`
	Proposed      bool     `json:"proposed,omitempty"`
}

type metaTypeKind string

const (
	metaTypeBase          metaTypeKind = "base"
	metaTypeReference     metaTypeKind = "reference"
	metaTypeArray         metaTypeKind = "array"
	metaTypeMap           metaTypeKind = "map"
	metaTypeOr            metaTypeKind = "or"
	metaTypeAnd           metaTypeKind = "and"
	metaTypeTuple         metaTypeKind = "tuple"
	metaTypeLiteral       metaTypeKind = "literal"
	metaTypeStringLiteral metaTypeKind = "stringLiteral"
)

type literalType struct {
	Properties []property `json:"properties"`
}

type metaType struct {
	Kind          metaTypeKind
	Name          string
	Element       *metaType
	Items         []metaType
	Key           *metaType
	Value         *metaType
	Literal       *literalType
	StringLiteral string
}

func (t *metaType) UnmarshalJSON(data []byte) error {
	var raw struct {
		Kind metaTypeKind `json:"kind"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*t = metaType{Kind: raw.Kind}

	switch raw.Kind {
	case metaTypeBase, metaTypeReference:
		var decoded struct {
			Kind metaTypeKind `json:"kind"`
			Name string       `json:"name"`
		}
		if err := json.Unmarshal(data, &decoded); err != nil {
			return err
		}
		t.Name = decoded.Name

	case metaTypeArray:
		var decoded struct {
			Kind    metaTypeKind `json:"kind"`
			Element metaType     `json:"element"`
		}
		if err := json.Unmarshal(data, &decoded); err != nil {
			return err
		}
		t.Element = &decoded.Element

	case metaTypeMap:
		var decoded struct {
			Kind  metaTypeKind `json:"kind"`
			Key   metaType     `json:"key"`
			Value metaType     `json:"value"`
		}
		if err := json.Unmarshal(data, &decoded); err != nil {
			return err
		}
		t.Key = &decoded.Key
		t.Value = &decoded.Value

	case metaTypeOr, metaTypeAnd, metaTypeTuple:
		var decoded struct {
			Kind  metaTypeKind `json:"kind"`
			Items []metaType   `json:"items"`
		}
		if err := json.Unmarshal(data, &decoded); err != nil {
			return err
		}
		t.Items = decoded.Items

	case metaTypeLiteral:
		var decoded struct {
			Kind  metaTypeKind `json:"kind"`
			Value literalType  `json:"value"`
		}
		if err := json.Unmarshal(data, &decoded); err != nil {
			return err
		}
		t.Literal = &decoded.Value

	case metaTypeStringLiteral:
		var decoded struct {
			Kind  metaTypeKind `json:"kind"`
			Value string       `json:"value"`
		}
		if err := json.Unmarshal(data, &decoded); err != nil {
			return err
		}
		t.StringLiteral = decoded.Value

	default:
		return fmt.Errorf("metaType: unsupported kind %q", raw.Kind)
	}

	return nil
}
