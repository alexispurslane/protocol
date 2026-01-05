// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type config struct {
	metaPath      string
	structMapPath string
	outDir        string
	packageName   string
	fallbackFile  string
	strict        bool
}

func main() {
	cfg := parseFlags()
	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseFlags() config {
	_, file, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(file)
	defaultOut := filepath.Clean(filepath.Join(baseDir, "..", ".."))

	structMapPath := filepath.Join(baseDir, "schema", "struct-map.json")
	metaPath := filepath.Join(baseDir, "schema", "metaModel.json")

	cfg := config{
		structMapPath: structMapPath,
		metaPath:      metaPath,
	}
	flag.StringVar(&cfg.structMapPath, "struct-map", structMapPath, "path to struct-map.json")
	flag.StringVar(&cfg.outDir, "out", defaultOut, "output directory for generated Go files")
	flag.StringVar(&cfg.packageName, "package", "protocol", "package name for generated files")
	flag.StringVar(&cfg.fallbackFile, "fallback-file", "lsp_gen.go", "file name for types not in struct-map")
	flag.BoolVar(&cfg.strict, "strict", true, "fail if struct-map entries are unmatched or if types are missing")
	flag.Parse()

	return cfg
}

func run(cfg config) error {
	model, err := loadMetaModel(cfg.metaPath)
	if err != nil {
		return err
	}

	entries, err := loadStructMap(cfg.structMapPath)
	if err != nil {
		return err
	}

	gen, err := newGenerator(model, entries, cfg)
	if err != nil {
		return err
	}

	files, err := gen.Generate()
	if err != nil {
		return err
	}

	return writeFiles(cfg.outDir, files)
}
