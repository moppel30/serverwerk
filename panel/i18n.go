package main

// Sprache des Panels. Der Code ist auf Deutsch geschrieben; für Englisch werden
// Meldungen über das Wörterbuch web/i18n/en.json übersetzt (dasselbe wie im Browser).

import (
	"encoding/json"
	"io/fs"
	"regexp"
	"sort"
	"sync/atomic"
	"unicode"
	"unicode/utf8"
)

var langCur atomic.Value // "de" oder "en"

type dict struct {
	m  map[string]string
	re *regexp.Regexp
}

var enDict *dict

func loadDict(web fs.FS) {
	b, err := fs.ReadFile(web, "i18n/en.json")
	if err != nil {
		return
	}
	m := map[string]string{}
	if json.Unmarshal(b, &m) != nil || len(m) == 0 {
		return
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Längste zuerst, damit ganze Sätze vor einzelnen Wörtern greifen.
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = regexp.QuoteMeta(k)
	}
	enDict = &dict{m: m, re: regexp.MustCompile(stringsJoin(parts, "|"))}
}

func stringsJoin(p []string, sep string) string {
	n := 0
	for _, s := range p {
		n += len(s) + len(sep)
	}
	b := make([]byte, 0, n)
	for i, s := range p {
		if i > 0 {
			b = append(b, sep...)
		}
		b = append(b, s...)
	}
	return string(b)
}

func setLang(l string) {
	if l != "en" {
		l = "de"
	}
	langCur.Store(l)
}

func lang() string {
	if v, ok := langCur.Load().(string); ok {
		return v
	}
	return "de"
}

func isLetter(r rune) bool { return unicode.IsLetter(r) }

// T übersetzt einen deutschen Text, wenn das Panel auf Englisch steht.
func T(s string) string {
	if lang() != "en" || enDict == nil || s == "" {
		return s
	}
	return translateWith(enDict, s)
}

func translateWith(d *dict, s string) string {
	idx := d.re.FindAllStringIndex(s, -1)
	if idx == nil {
		return s
	}
	out := make([]byte, 0, len(s)+32)
	last := 0
	for _, m := range idx {
		a, b := m[0], m[1]
		key := s[a:b]
		first, _ := utf8.DecodeRuneInString(key)
		lastR, _ := utf8.DecodeLastRuneInString(key)
		before, _ := utf8.DecodeLastRuneInString(s[:a])
		after, _ := utf8.DecodeRuneInString(s[b:])
		// Nur ganze Wörter ersetzen ("Port" nicht in "Portal").
		if (isLetter(first) && a > 0 && isLetter(before)) || (isLetter(lastR) && b < len(s) && isLetter(after)) {
			continue
		}
		out = append(out, s[last:a]...)
		out = append(out, d.m[key]...)
		last = b
	}
	out = append(out, s[last:]...)
	return string(out)
}
