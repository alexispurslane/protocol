// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	files := map[string][]byte{
		"a.go": []byte("first"),
	}
	if err := writeFiles(dir, files); err != nil {
		t.Fatalf("writeFiles error: %v", err)
	}
	path := filepath.Join(dir, "a.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file error: %v", err)
	}
	if string(data) != "first" {
		t.Fatalf("unexpected content: %s", data)
	}

	files["a.go"] = []byte("second")
	if err := writeFiles(dir, files); err != nil {
		t.Fatalf("writeFiles second error: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file error: %v", err)
	}
	if string(data) != "second" {
		t.Fatalf("unexpected content after update: %s", data)
	}
}
