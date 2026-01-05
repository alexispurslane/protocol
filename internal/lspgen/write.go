// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func writeFiles(outDir string, files map[string][]byte) error {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, rel := range paths {
		content := files[rel]
		path := filepath.Join(outDir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create dir for %s: %w", path, err)
		}
		if existing, err := os.ReadFile(path); err == nil {
			if bytes.Equal(existing, content) {
				continue
			}
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	return nil
}
