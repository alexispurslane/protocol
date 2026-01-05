// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestParseFlags(t *testing.T) {
	t.Parallel()

	origArgs := os.Args
	origCommand := flag.CommandLine
	t.Cleanup(func() {
		os.Args = origArgs
		flag.CommandLine = origCommand
	})

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{
		"cmd",
		"-meta", "meta.json",
		"-struct-map", "struct-map.json",
		"-out", "out",
		"-package", "proto",
		"-fallback-file", "gen.go",
		"-strict",
	}

	cfg := parseFlags()
	if cfg.metaPath != "meta.json" || cfg.structMapPath != "struct-map.json" || cfg.outDir != "out" || cfg.packageName != "proto" || cfg.fallbackFile != "gen.go" || !cfg.strict {
		t.Fatalf("parseFlags mismatch: %#v", cfg)
	}
}

func TestRun(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	metaPath := filepath.Join(tmp, "meta.json")
	structPath := filepath.Join(tmp, "struct-map.json")

	meta := `{"metaData":{"version":"1.0"},"requests":[],"notifications":[],"structures":[{"name":"Foo","properties":[]}],"enumerations":[],"typeAliases":[]}`
	if err := os.WriteFile(metaPath, []byte(meta), 0o644); err != nil {
		t.Fatalf("write meta error: %v", err)
	}
	structMap := `[{"name":"Foo","file":"foo.go"}]`
	if err := os.WriteFile(structPath, []byte(structMap), 0o644); err != nil {
		t.Fatalf("write struct map error: %v", err)
	}

	cfg := config{metaPath: metaPath, structMapPath: structPath, outDir: tmp, packageName: "protocol", fallbackFile: "lsp_gen.go"}
	if err := run(cfg); err != nil {
		t.Fatalf("run error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(tmp, "foo.go"))
	if err != nil {
		t.Fatalf("read output error: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected output content")
	}
}
