// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"os"

	"github.com/go-json-experiment/json"
)

type structMapEntry struct {
	Name string `json:"name"`
	File string `json:"file"`
}

func loadMetaModel(path string) (metaModel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return metaModel{}, fmt.Errorf("read meta model: %w", err)
	}

	var model metaModel
	if err := json.Unmarshal(data, &model); err != nil {
		return metaModel{}, fmt.Errorf("decode meta model: %w", err)
	}

	return model, nil
}

func loadStructMap(path string) ([]structMapEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read struct map: %w", err)
	}

	var entries []structMapEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("decode struct map: %w", err)
	}

	return entries, nil
}
