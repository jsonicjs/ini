/* Copyright (c) 2021-2025 Richard Rodger, MIT License */

package ini

import (
	jsonic "github.com/jsonicjs/jsonic/go"
)

// IniOptions configures the INI parser.
type IniOptions struct {
	Multiline *MultilineOptions
	Section   *SectionOptions
	Comment   *CommentOptions
}

// MultilineOptions controls multiline value continuation.
type MultilineOptions struct {
	// Continuation character before newline. Default: "\\".
	// Set to empty string to disable backslash continuation.
	Continuation *string
	// When true, indented continuation lines extend the previous value.
	Indent *bool
}

// SectionOptions controls section header handling.
type SectionOptions struct {
	// How to handle duplicate section headers.
	// "merge" (default): combine keys from all occurrences.
	// "override": last section occurrence replaces earlier ones.
	// "error": throw when a previously declared section header appears again.
	Duplicate string
}

// CommentOptions controls comment behavior.
type CommentOptions struct {
	Inline *InlineCommentOptions
}

// InlineCommentOptions controls inline comment behavior.
type InlineCommentOptions struct {
	// Whether inline comments are active. Default: false.
	Active *bool
	// Characters that start an inline comment. Default: ["#", ";"].
	Chars []string
	// Escape mechanisms for literal comment characters in values.
	Escape *InlineEscapeOptions
}

// InlineEscapeOptions controls escaping of inline comment characters.
type InlineEscapeOptions struct {
	// Allow \; and \# to produce literal ; and #. Default: true.
	Backslash *bool
	// Require whitespace before comment char to trigger. Default: false.
	Whitespace *bool
}

// resolved holds fully resolved options with defaults applied.
type resolved struct {
	multiline     bool
	continuation  string // "" means disabled
	indent        bool
	dupSection    string
	inlineActive  bool
	inlineChars   map[rune]bool
	inlineCharStr []string
	escBackslash  bool
	escWhitespace bool
}

func boolOpt(p *bool, def bool) bool {
	if p != nil {
		return *p
	}
	return def
}

func stringOpt(p *string, def string) string {
	if p != nil {
		return *p
	}
	return def
}

func resolve(o *IniOptions) *resolved {
	r := &resolved{
		dupSection:    "merge",
		inlineChars:   map[rune]bool{'#': true, ';': true},
		inlineCharStr: []string{"#", ";"},
		escBackslash:  true,
	}

	if o.Multiline != nil {
		r.multiline = true
		r.continuation = stringOpt(o.Multiline.Continuation, "\\")
		r.indent = boolOpt(o.Multiline.Indent, false)
	}

	if o.Section != nil && o.Section.Duplicate != "" {
		r.dupSection = o.Section.Duplicate
	}

	if o.Comment != nil && o.Comment.Inline != nil {
		ic := o.Comment.Inline
		r.inlineActive = boolOpt(ic.Active, false)
		if ic.Chars != nil && len(ic.Chars) > 0 {
			r.inlineChars = make(map[rune]bool)
			r.inlineCharStr = ic.Chars
			for _, s := range ic.Chars {
				if len(s) > 0 {
					r.inlineChars[rune(s[0])] = true
				}
			}
		}
		if ic.Escape != nil {
			r.escBackslash = boolOpt(ic.Escape.Backslash, true)
			r.escWhitespace = boolOpt(ic.Escape.Whitespace, false)
		}
	}

	return r
}

// Parse parses an INI string and returns a map.
func Parse(src string, opts ...IniOptions) (map[string]any, error) {
	var o IniOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	j := MakeJsonic(o)
	result, err := j.Parse(src)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return map[string]any{}, nil
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return map[string]any{}, nil
}

// MakeJsonic creates a jsonic instance configured for INI parsing.
func MakeJsonic(opts ...IniOptions) *jsonic.Jsonic {
	var o IniOptions
	if len(opts) > 0 {
		o = opts[0]
	}

	r := resolve(&o)

	bTrue := true
	bFalse := false

	jopts := jsonic.Options{
		Rule: &jsonic.RuleOptions{
			Start: "ini",
		},
		Number: &jsonic.NumberOptions{
			Lex: &bFalse,
		},
		Value: &jsonic.ValueOptions{
			Lex: &bTrue,
		},
		Comment: &jsonic.CommentOptions{
			Lex: &bTrue,
			Def: map[string]*jsonic.CommentDef{
				"hash": {Line: true, Start: "#"},
				"semi": {Line: true, Start: ";"},
			},
		},
		String: &jsonic.StringOptions{
			Lex:   &bTrue,
			Chars: `'"`,
		},
		Text: &jsonic.TextOptions{
			Lex: &bFalse,
		},
		Lex: &jsonic.LexOptions{
			EmptyResult: map[string]any{},
		},
	}

	j := jsonic.Make(jopts)

	pluginMap := optionsToMap(&o, r)
	j.Use(Ini, pluginMap)

	return j
}

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

// optionsToMap converts IniOptions to a map for the plugin interface.
func optionsToMap(o *IniOptions, r *resolved) map[string]any {
	m := make(map[string]any)
	m["_resolved"] = r
	return m
}

// mapToOptions extracts resolved options from the plugin map.
func mapToResolved(m map[string]any) *resolved {
	if m == nil {
		return resolve(&IniOptions{})
	}
	if r, ok := m["_resolved"].(*resolved); ok {
		return r
	}
	return resolve(&IniOptions{})
}
