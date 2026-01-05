// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoadMetaModel(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "meta.json")
	content := `{"metaData":{"version":"1.0"},"requests":[],"notifications":[],"structures":[],"enumerations":[],"typeAliases":[]}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write meta file: %v", err)
	}

	got, err := loadMetaModel(path)
	if err != nil {
		t.Fatalf("loadMetaModel error: %v", err)
	}
	if diff := cmp.Diff("1.0", got.MetaData.Version); diff != "" {
		t.Errorf("version mismatch (-want +got):\n%s", diff)
	}
}

func TestLoadMetaModelError(t *testing.T) {
	t.Parallel()

	_, err := loadMetaModel("/non-existent/meta.json")
	if err == nil {
		t.Fatalf("expected error for missing meta model file")
	}
}

func TestLoadStructMap(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "map.json")
	content := `[{"name":"Foo","file":"foo.go"}]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write struct map: %v", err)
	}

	got, err := loadStructMap(path)
	if err != nil {
		t.Fatalf("loadStructMap error: %v", err)
	}
	want := []structMapEntry{{Name: "Foo", File: "foo.go"}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("struct map mismatch (-want +got):\n%s", diff)
	}
}

func TestLoadStructMapError(t *testing.T) {
	t.Parallel()

	_, err := loadStructMap("/non-existent/struct-map.json")
	if err == nil {
		t.Fatalf("expected error for missing struct map file")
	}
}
