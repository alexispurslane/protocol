// Copyright 2026 The Go Language Server Authors.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"strings"
)

var initialisms = map[string]struct{}{
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
	"LSP":   {},
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
}

func goName(name string) string {
	words := splitWords(name)
	if len(words) == 0 {
		return name
	}

	var b strings.Builder
	for _, word := range words {
		upper := strings.ToUpper(word)
		if _, ok := initialisms[upper]; ok {
			b.WriteString(upper)
			continue
		}
		b.WriteString(strings.ToUpper(word[:1]))
		if len(word) > 1 {
			b.WriteString(word[1:])
		}
	}

	return b.String()
}

func splitWords(name string) []string {
	if name == "" {
		return nil
	}

	words := make([]string, 0, len(name))
	start := 0
	for i := 1; i < len(name); i++ {
		curr := name[i]
		prev := name[i-1]
		next := byte(0)
		if i+1 < len(name) {
			next = name[i+1]
		}
		if isUpper(curr) && (!isUpper(prev) || (next != 0 && !isUpper(next))) {
			words = append(words, name[start:i])
			start = i
		}
	}

	words = append(words, name[start:])
	return words
}

func isUpper(b byte) bool {
	return b >= 'A' && b <= 'Z'
}

func pluralize(name string) string {
	if name == "" {
		return name
	}

	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "y") && len(name) > 1 {
		prev := lower[len(lower)-2]
		if prev < 'a' || prev > 'z' || !strings.ContainsRune("aeiou", rune(prev)) {
			return name[:len(name)-1] + "ies"
		}
	}

	if strings.HasSuffix(lower, "s") || strings.HasSuffix(lower, "x") || strings.HasSuffix(lower, "z") || strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "sh") {
		return name + "es"
	}

	return name + "s"
}

func jsonTag(name string, optional, omitEmpty bool) string {
	if !optional {
		return "json:\"" + name + "\""
	}
	if omitEmpty {
		return "json:\"" + name + ",omitempty\""
	}
	return "json:\"" + name + ",omitzero\""
}
