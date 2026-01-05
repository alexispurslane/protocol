// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

func (g *generator) computeReachable() map[string]struct{} {
	reachable := make(map[string]struct{})
	addName := func(name string) {
		if name == "" || isSkippedType(name) {
			return
		}
		reachable[name] = struct{}{}
	}

	for name := range g.registry.structs {
		addName(name)
	}
	for name := range g.registry.enums {
		addName(name)
	}
	for name := range g.registry.aliases {
		addName(name)
	}

	if len(reachable) == 0 {
		return nil
	}

	return reachable
}

func (g *generator) isReachable(name string) bool {
	if g.reachable == nil {
		return true
	}
	_, ok := g.reachable[name]
	return ok
}
