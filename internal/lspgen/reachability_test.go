// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestComputeReachable(t *testing.T) {
	t.Parallel()

	model := metaModel{
		Structures: []structure{
			{Name: "Foo"},
			{Name: "Bar"},
			{Name: "Baz"},
		},
		TypeAliases: []typeAlias{
			{Name: "LSPAny", Type: metaType{Kind: metaTypeBase, Name: "string"}},
		},
	}
	gen, err := newGenerator(model, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}

	if diff := cmp.Diff(true, gen.isReachable("Foo")); diff != "" {
		t.Fatalf("expected Foo reachable (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, gen.isReachable("Bar")); diff != "" {
		t.Fatalf("expected Bar reachable (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, gen.isReachable("Baz")); diff != "" {
		t.Fatalf("expected Baz reachable (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, gen.isReachable("LSPAny")); diff != "" {
		t.Fatalf("expected LSPAny unreachable (-want +got):\n%s", diff)
	}
}

func TestIsReachableDefaultsToTrue(t *testing.T) {
	t.Parallel()

	model := metaModel{
		Structures: []structure{{Name: "Foo"}},
	}
	gen, err := newGenerator(model, nil, config{})
	if err != nil {
		t.Fatalf("newGenerator error: %v", err)
	}
	if diff := cmp.Diff(true, gen.isReachable("Foo")); diff != "" {
		t.Fatalf("expected reachable by default (-want +got):\n%s", diff)
	}
}
