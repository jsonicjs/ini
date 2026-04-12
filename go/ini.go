/* Copyright (c) 2021-2025 Richard Rodger, MIT License */

package ini

import (
	"fmt"
	"strings"

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
	j.Use(iniPlugin, pluginMap)

	return j
}

// --- BEGIN EMBEDDED ini-grammar.jsonic ---
const grammarText = `
# INI Grammar Definition
# Parsed by a standard Jsonic instance and passed to jsonic.grammar()
# Function references (@ prefixed) are resolved against the refs map

{
  options: rule: { start: ini exclude: jsonic }
  options: lex: { emptyResult: {} }
  options: fixed: token: { '#EQ': '=' '#DOT': '.' '#OB': null '#CB': null '#CL': null }
  options: line: { check: '@line-check' }
  options: number: { lex: false }
  options: string: { lex: true chars: QUOTE_CHARS abandon: true }
  options: text: { lex: false }
  options: comment: def: {
    hash: { eatline: true }
    slash: null
    multi: null
    semi: { line: true start: ';' lex: true eatline: true }
  }

  rule: ini: open: [
    { s: '#OS' p: table b: 1 }
    { s: ['#HK #ST #VL' '#EQ'] p: table b: 2 }
    { s: ['#HV' '#OS'] p: table b: 2 }
    { s: '#ZZ' }
  ]

  rule: table: open: [
    { s: '#OS' p: dive }
    { s: ['#HK #ST #VL' '#EQ'] p: map b: 2 }
    { s: ['#HV' '#OS'] p: map b: 2 }
    { s: '#CS' p: map }
    { s: '#ZZ' }
  ]
  rule: table: close: [
    { s: '#OS' r: table b: 1 }
    { s: '#CS' r: table a: '@table-close-dive' }
    { s: '#ZZ' }
  ]

  rule: dive: open: [
    { s: ['#DK' '#DOT'] a: '@dive-push' p: dive }
    { s: '#DK' a: '@dive-push' }
  ]
  rule: dive: close: [
    { s: '#CS' b: 1 }
  ]

  rule: map: open: {
    alts: [
      { s: ['#HK #ST #VL' '#EQ'] c: '@is-table-parent' p: pair b: 2 }
      { s: ['#HK #ST #VL'] c: '@is-table-parent' p: pair b: 1 }
    ]
    inject: { append: true }
  }
  rule: map: close: [
    { s: '#OS' b: 1 }
    { s: '#ZZ' }
  ]

  rule: pair: open: [
    { s: ['#HK #ST #VL' '#EQ'] c: '@is-table-grandparent' p: val a: '@pair-key-eq' }
    { s: '#HK' c: '@is-table-grandparent' a: '@pair-key-bool' }
  ]
  rule: pair: close: [
    { s: ['#HK #ST #VL' '#CL'] c: '@is-table-grandparent' e: '@pair-close-err' }
    { s: ['#HK #ST #VL'] b: 1 r: pair }
    { s: '#OS' b: 1 }
  ]
}
`
// --- END EMBEDDED ini-grammar.jsonic ---

// iniPlugin is the jsonic plugin that adds INI parsing support.
func iniPlugin(j *jsonic.Jsonic, pluginOpts map[string]any) {
	opts := mapToResolved(pluginOpts)

	cfg := j.Config()

	// Disable JSON structure tokens except [ and ].
	delete(cfg.FixedTokens, "{")
	delete(cfg.FixedTokens, "}")
	delete(cfg.FixedTokens, ":")
	cfg.SortFixedTokens()

	// Register custom fixed tokens.
	j.Token("#EQ", "=")
	j.Token("#DOT", ".")
	cfg.SortFixedTokens()

	// Register custom token types used by matchers.
	HK := j.Token("#HK") // Hoover Key
	HV := j.Token("#HV") // Hoover Value
	DK := j.Token("#DK") // Dive Key (section path part)
	ST := j.Token("#ST") // String
	OS := j.Token("#OS") // [
	CS := j.Token("#CS") // ]
	ZZ := j.Token("#ZZ")

	// Ensure these exist for grammar file token resolution.
	j.Token("#VL")
	j.Token("#CL")

	// Exclude default jsonic grammar rules.
	j.Exclude("jsonic", "imp")

	// Set start rule.
	cfg.RuleStart = "ini"

	// Disable text lexing (we handle it with custom matchers).
	cfg.TextLex = false

	// Add = to ender chars so the built-in matchers don't consume past it.
	if cfg.EnderChars == nil {
		cfg.EnderChars = make(map[rune]bool)
	}
	cfg.EnderChars['='] = true
	cfg.EnderChars['['] = true
	cfg.EnderChars[']'] = true
	cfg.EnderChars['.'] = true

	// ---- Custom Matchers ----
	// Context is determined from the current rule (like TS Hoover), not
	// from a mutable mode variable. This keeps the matchers stateless and
	// lets the grammar file drive all rule definitions without Go-specific
	// action wrappers.

	// Key matcher: reads a key token (until =, newline, [, ], or EOF).
	// Fires when the current rule is NOT "dive" and NOT in value context
	// (parent rule is not "pair"), matching Hoover's key block conditions:
	//   start.rule.current.exclude: ['dive'], state: 'oc'
	j.AddMatcher("inikey", 100000, func(lex *jsonic.Lex, rule *jsonic.Rule) *jsonic.Token {
		if rule != nil {
			if rule.Name == "dive" {
				return nil
			}
			if rule.Parent != nil && rule.Parent.Name == "pair" {
				return nil
			}
		}

		pnt := lex.Cursor()
		src := lex.Src
		sI := pnt.SI
		if sI >= pnt.Len {
			return nil
		}

		// Pass through to built-in matchers for non-key chars.
		ch := src[sI]
		if ch == '=' || ch == '.' ||
			ch == '#' || ch == ';' ||
			ch == '"' || ch == '\'' ||
			ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			return nil
		}
		if ch == '[' {
			return nil
		}
		if ch == ']' {
			return nil
		}

		startI := sI
		for sI < pnt.Len {
			c := src[sI]
			if c == '=' || c == '\n' || c == '\r' {
				break
			}
			// Allow [] in key for array syntax.
			if c == '[' {
				if sI+1 < pnt.Len && src[sI+1] == ']' {
					sI += 2
					continue
				}
				break
			}
			if c == ']' {
				break
			}
			// Inline comment chars in key position.
			if opts.inlineActive && opts.inlineChars[rune(c)] {
				break
			}
			// Escape handling in keys.
			if c == '\\' && sI+1 < pnt.Len {
				next := src[sI+1]
				if next == '.' || next == ']' || next == '[' || next == '\\' {
					sI += 2
					continue
				}
				if opts.inlineActive && opts.escBackslash && opts.inlineChars[rune(next)] {
					sI += 2
					continue
				}
			}
			sI++
		}

		if sI == startI {
			return nil
		}

		raw := src[startI:sI]
		val := processKeyEscapes(strings.TrimSpace(raw))

		tkn := lex.Token("#HK", HK, val, raw)
		pnt.SI = sI
		pnt.CI += sI - startI
		return tkn
	})

	// Value matcher: reads a value token (until newline, comment, or EOF).
	// Fires when the parent rule is "pair" (value context), matching
	// Hoover's endofline block condition: start.rule.parent.include: ['pair', 'elem']
	j.AddMatcher("inival", 100001, func(lex *jsonic.Lex, rule *jsonic.Rule) *jsonic.Token {
		if rule == nil || rule.Parent == nil ||
			(rule.Parent.Name != "pair" && rule.Parent.Name != "elem") {
			return nil
		}

		pnt := lex.Cursor()
		src := lex.Src
		sI := pnt.SI
		rI := pnt.RI
		cI := pnt.CI
		if sI >= pnt.Len {
			// EOF: empty value.
			return nil
		}

		// Skip leading whitespace (since we run before the space matcher).
		for sI < pnt.Len && (src[sI] == ' ' || src[sI] == '\t') {
			sI++
			cI++
		}

		// Check for immediate newline → empty value.
		if sI >= pnt.Len || src[sI] == '\n' || src[sI] == '\r' {
			// Consume the newline if present.
			if sI < pnt.Len {
				if src[sI] == '\r' && sI+1 < pnt.Len && src[sI+1] == '\n' {
					sI += 2
				} else {
					sI++
				}
				rI++
				cI = 1
			}
			tkn := lex.Token("#HV", HV, "", src[pnt.SI:sI])
			pnt.SI = sI
			pnt.RI = rI
			pnt.CI = cI
			return tkn
		}

		// Check for line-start comment chars → empty value.
		ch := src[sI]
		if ch == '#' || ch == ';' {
			tkn := lex.Token("#HV", HV, "", src[pnt.SI:sI])
			pnt.SI = sI
			pnt.RI = rI
			pnt.CI = cI
			return tkn
		}

		// Don't match at quotes (let string matcher handle them).
		if ch == '"' || ch == '\'' {
			// Advance past the whitespace we skipped.
			pnt.SI = sI
			pnt.CI = cI
			return nil
		}
		// Don't match at [ or ] (let fixed token matcher handle them).
		if ch == '[' || ch == ']' {
			pnt.SI = sI
			pnt.CI = cI
			return nil
		}

		startI := pnt.SI // Include leading whitespace in src
		var chars []byte

		for sI < pnt.Len {
			c := src[sI]

			// Check for inline comment characters.
			if opts.inlineActive && opts.inlineChars[rune(c)] {
				if opts.escWhitespace {
					// Only treat as comment if preceded by whitespace.
					if len(chars) > 0 && (chars[len(chars)-1] == ' ' || chars[len(chars)-1] == '\t') {
						break
					}
					// Not preceded by whitespace: treat as literal.
					chars = append(chars, c)
					sI++
					cI++
					continue
				}
				break
			}

			// Check for backslash continuation before newline.
			if opts.multiline && opts.continuation != "" && c == opts.continuation[0] {
				if sI+1 < pnt.Len && src[sI+1] == '\n' {
					sI += 2
					rI++
					cI = 1
					for sI < pnt.Len && (src[sI] == ' ' || src[sI] == '\t') {
						sI++
						cI++
					}
					continue
				}
				if sI+2 < pnt.Len && src[sI+1] == '\r' && src[sI+2] == '\n' {
					sI += 3
					rI++
					cI = 1
					for sI < pnt.Len && (src[sI] == ' ' || src[sI] == '\t') {
						sI++
						cI++
					}
					continue
				}
			}

			// Check for newline.
			if c == '\n' || (c == '\r' && sI+1 < pnt.Len && src[sI+1] == '\n') {
				// Indent continuation.
				if opts.multiline && opts.indent {
					var nextI int
					if c == '\r' {
						nextI = sI + 2
					} else {
						nextI = sI + 1
					}
					if nextI < pnt.Len && (src[nextI] == ' ' || src[nextI] == '\t') {
						rI++
						cI = 1
						sI = nextI
						for sI < pnt.Len && (src[sI] == ' ' || src[sI] == '\t') {
							sI++
							cI++
						}
						chars = append(chars, ' ')
						continue
					}
				}

				// Normal newline: end value and consume it.
				if c == '\r' {
					sI += 2
				} else {
					sI++
				}
				rI++
				cI = 1
				break
			}

			// Bare \r without \n.
			if c == '\r' {
				sI++
				rI++
				cI = 1
				break
			}

			// Handle escape sequences.
			if c == '\\' && sI+1 < pnt.Len {
				next := src[sI+1]
				if opts.inlineActive && opts.escBackslash && opts.inlineChars[rune(next)] {
					chars = append(chars, next)
					sI += 2
					cI += 2
					continue
				}
				if next == '\\' {
					chars = append(chars, '\\')
					sI += 2
					cI += 2
					continue
				}
			}

			chars = append(chars, c)
			sI++
			cI++
		}

		valStr := strings.TrimSpace(string(chars))
		val := resolveValue(valStr)

		tkn := lex.Token("#HV", HV, val, src[startI:sI])
		pnt.SI = sI
		pnt.RI = rI
		pnt.CI = cI
		return tkn
	})

	// Dive key matcher: reads section path parts (until ] or .).
	// Fires only when the current rule is "dive", matching Hoover's
	// divekey block condition: start.rule.current.include: ['dive']
	j.AddMatcher("inidive", 100002, func(lex *jsonic.Lex, rule *jsonic.Rule) *jsonic.Token {
		if rule == nil || rule.Name != "dive" {
			return nil
		}

		pnt := lex.Cursor()
		src := lex.Src
		sI := pnt.SI
		if sI >= pnt.Len {
			return nil
		}

		// Pass through for fixed tokens and whitespace.
		ch := src[sI]
		if ch == ']' || ch == '.' || ch == '[' ||
			ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			return nil
		}

		startI := sI
		for sI < pnt.Len {
			c := src[sI]
			if c == ']' || c == '.' {
				break
			}
			if c == '\n' || c == '\r' {
				break
			}
			if c == '\\' && sI+1 < pnt.Len {
				next := src[sI+1]
				if next == ']' || next == '.' || next == '\\' {
					sI += 2
					continue
				}
			}
			sI++
		}

		if sI == startI {
			return nil
		}

		raw := src[startI:sI]
		val := processDiveEscapes(strings.TrimSpace(raw))

		tkn := lex.Token("#DK", DK, val, raw)
		pnt.SI = sI
		pnt.CI += sI - startI
		return tkn
	})

	// ---- Grammar Rules ----
	// Rules ini, table, dive, map, pair are loaded from ini-grammar.jsonic
	// via j.Grammar(), mirroring the TS approach. State actions use the
	// @rulename-bo/bc/ac naming convention for auto-wiring.
	// The val rule is defined in Go code (needs custom open alts and
	// complex AC handler not expressible in the grammar file).

	var declaredSections map[string]bool

	// Function refs (matching @ names in the grammar file).
	// State actions (@ini-bo, @table-bo, @table-bc) are auto-wired by Grammar().
	refs := map[jsonic.FuncRef]any{
		// State actions.
		"@ini-bo": jsonic.StateAction(func(r *jsonic.Rule, ctx *jsonic.Context) {
			r.Node = make(map[string]any)
			declaredSections = make(map[string]bool)
		}),

		"@table-bo": jsonic.StateAction(func(r *jsonic.Rule, ctx *jsonic.Context) {
			r.Node = r.Parent.Node

			if r.Prev != nil && r.Prev != jsonic.NoRule {
				if dive, ok := r.Prev.U["dive"].([]string); ok && len(dive) > 0 {
					sectionKey := strings.Join(dive, "\x00")
					isDuplicate := declaredSections[sectionKey]

					if isDuplicate && opts.dupSection == "error" {
						panic(fmt.Sprintf("Duplicate section: [%s]", strings.Join(dive, ".")))
					}

					node, _ := r.Node.(map[string]any)
					for dI := 0; dI < len(dive); dI++ {
						if dI == len(dive)-1 && isDuplicate && opts.dupSection == "override" {
							newSection := make(map[string]any)
							node[dive[dI]] = newSection
							node = newSection
						} else {
							if existing, ok := node[dive[dI]].(map[string]any); ok {
								node = existing
							} else {
								newSection := make(map[string]any)
								node[dive[dI]] = newSection
								node = newSection
							}
						}
					}
					r.Node = node
					declaredSections[sectionKey] = true
				}
			}
		}),

		"@table-bc": jsonic.StateAction(func(r *jsonic.Rule, ctx *jsonic.Context) {
			if childMap, ok := r.Child.Node.(map[string]any); ok {
				if nodeMap, ok := r.Node.(map[string]any); ok {
					for k, v := range childMap {
						nodeMap[k] = v
					}
				}
			}
		}),

		// Alt actions.
		"@table-close-dive": jsonic.AltAction(func(r *jsonic.Rule, ctx *jsonic.Context) {
			if r.Child != nil && r.Child != jsonic.NoRule {
				if dive, ok := r.Child.U["dive"].([]string); ok {
					r.U["dive"] = dive
				}
			}
		}),

		"@dive-push": jsonic.AltAction(func(r *jsonic.Rule, ctx *jsonic.Context) {
			dive := getDive(r.Parent)
			val, _ := r.O0.Val.(string)
			dive = append(dive, val)
			r.U["dive"] = dive
			if r.Parent != nil && r.Parent != jsonic.NoRule {
				r.Parent.U["dive"] = dive
			}
		}),

		"@pair-key-eq": jsonic.AltAction(func(r *jsonic.Rule, ctx *jsonic.Context) {
			key := tokenString(r.O0)
			nodeMap, _ := r.Node.(map[string]any)
			if nodeMap == nil {
				return
			}

			if _, isArr := nodeMap[key].([]any); isArr {
				r.U["ini_array"] = nodeMap[key]
			} else if len(key) > 2 && strings.HasSuffix(key, "[]") {
				arrayKey := key[:len(key)-2]
				r.U["key"] = arrayKey
				if existing, ok := nodeMap[arrayKey].([]any); ok {
					r.U["ini_array"] = existing
				} else if _, exists := nodeMap[arrayKey]; exists {
					r.U["ini_array"] = []any{nodeMap[arrayKey]}
					nodeMap[arrayKey] = r.U["ini_array"]
				} else {
					arr := make([]any, 0)
					nodeMap[arrayKey] = arr
					r.U["ini_array"] = arr
				}
			} else {
				r.U["key"] = key
				r.U["pair"] = true
			}
		}),

		"@pair-key-bool": jsonic.AltAction(func(r *jsonic.Rule, ctx *jsonic.Context) {
			key := tokenString(r.O0)
			if key != "" {
				if nodeMap, ok := r.Parent.Node.(map[string]any); ok {
					nodeMap[key] = true
				}
			}
		}),

		"@pair-close-err": jsonic.AltError(func(r *jsonic.Rule, ctx *jsonic.Context) *jsonic.Token {
			// Not used in Go (CL token is disabled).
			return nil
		}),

		"@val-empty": jsonic.AltAction(func(r *jsonic.Rule, ctx *jsonic.Context) {
			r.Node = ""
		}),

		// Conditions.
		"@is-table-parent": jsonic.AltCond(func(r *jsonic.Rule, ctx *jsonic.Context) bool {
			return r.Parent != nil && r.Parent.Name == "table"
		}),

		"@is-table-grandparent": jsonic.AltCond(func(r *jsonic.Rule, ctx *jsonic.Context) bool {
			return r.Parent != nil && r.Parent.Parent != nil &&
				r.Parent.Parent.Name == "table"
		}),
	}

	// Parse grammar file and apply rules via j.Grammar() — same as TS approach.
	parser := jsonic.Make()
	parsed, err := parser.Parse(grammarText)
	if err != nil {
		panic("failed to parse ini grammar: " + err.Error())
	}
	grammarDef := mapToGrammarSpec(parsed.(map[string]any), refs)
	if err := j.Grammar(grammarDef); err != nil {
		panic("failed to apply ini grammar: " + err.Error())
	}

	// ---- val rule ----
	// Defined in Go code: needs custom open alts and complex AC handler
	// not expressible in the declarative grammar file.
	j.Rule("val", func(rs *jsonic.RuleSpec) {
		rs.Clear()

		rs.AddBO(func(r *jsonic.Rule, ctx *jsonic.Context) {
			r.Node = jsonic.Undefined
		})

		rs.Open = []*jsonic.AltSpec{
			// Bracket chars at start of value: concat with next value.
			{S: [][]jsonic.Tin{{OS}}, R: "val",
				U: map[string]any{"ini_prev": true}},
			{S: [][]jsonic.Tin{{CS}}, R: "val",
				U: map[string]any{"ini_prev": true}},
			// String value.
			{S: [][]jsonic.Tin{{ST}}},
			// Hoover value (unquoted text).
			{S: [][]jsonic.Tin{{HV}}},
			// End of input: empty value.
			{S: [][]jsonic.Tin{{ZZ}},
				A: func(r *jsonic.Rule, ctx *jsonic.Context) {
					r.Node = ""
				}},
		}

		rs.AddAC(func(r *jsonic.Rule, ctx *jsonic.Context) {
			// Resolve value.
			if jsonic.IsUndefined(r.Node) || r.Node == nil {
				if r.O0 != nil && !r.O0.IsNoToken() {
					r.Node = resolveTokenVal(r.O0)
				} else {
					r.Node = ""
				}
			}

			// Handle single-quoted JSON parsing.
			if r.O0 != nil && r.O0.Tin == ST && len(r.O0.Src) > 0 && r.O0.Src[0] == '\'' {
				if s, ok := r.Node.(string); ok {
					r.Node = tryParseJSON(s)
				}
			}

			// Handle ini_prev concatenation.
			if r.Prev != nil && r.Prev != jsonic.NoRule {
				if _, ok := r.Prev.U["ini_prev"]; ok {
					valStr := fmt.Sprintf("%v", r.Node)
					r.Node = r.Prev.O0.Src + valStr
					r.Prev.Node = r.Node
					return
				}
			}

			// Handle array push.
			if r.Parent != nil && r.Parent != jsonic.NoRule {
				if arr, ok := r.Parent.U["ini_array"].([]any); ok {
					arr = append(arr, r.Node)
					r.Parent.U["ini_array"] = arr
					if key, ok := r.Parent.U["key"].(string); ok {
						if nodeMap, ok := r.Parent.Node.(map[string]any); ok {
							nodeMap[key] = arr
						}
					}
					return
				}
			}

			// Normal pair assignment.
			if r.Parent != nil && r.Parent != jsonic.NoRule {
				if key, ok := r.Parent.U["key"].(string); ok {
					if _, isPair := r.Parent.U["pair"]; isPair {
						if nodeMap, ok := r.Parent.Node.(map[string]any); ok {
							nodeMap[key] = r.Node
						}
					}
				}
			}
		})
	})
}

// ---- Helper functions ----

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

func optionsToMap(o *IniOptions, r *resolved) map[string]any {
	m := make(map[string]any)
	m["_resolved"] = r
	return m
}

func mapToResolved(m map[string]any) *resolved {
	if m == nil {
		return resolve(&IniOptions{})
	}
	if r, ok := m["_resolved"].(*resolved); ok {
		return r
	}
	return resolve(&IniOptions{})
}

func getDive(r *jsonic.Rule) []string {
	if r == nil || r == jsonic.NoRule {
		return nil
	}
	if dive, ok := r.U["dive"].([]string); ok {
		return dive
	}
	return nil
}

func tokenString(t *jsonic.Token) string {
	if t == nil || t.IsNoToken() {
		return ""
	}
	if s, ok := t.Val.(string); ok {
		return s
	}
	return t.Src
}

func resolveTokenVal(t *jsonic.Token) any {
	if !jsonic.IsUndefined(t.Val) {
		return t.Val
	}
	return t.Src
}

func processKeyEscapes(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			next := s[i+1]
			if next == '.' || next == ']' || next == '[' || next == '\\' {
				b.WriteByte(next)
				i++
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func processDiveEscapes(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			next := s[i+1]
			if next == ']' || next == '.' || next == '\\' {
				b.WriteByte(next)
				i++
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func tryParseJSON(s string) any {
	trimmed := strings.TrimSpace(s)
	switch trimmed {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	return s
}

func resolveValue(s string) any {
	switch s {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	return s
}

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

// mapToGrammarSpec converts a parsed grammar map (from jsonic.Parse of the
// grammar file) into a typed GrammarSpec. Only the "rule" section is used;
// options are set separately via MakeJsonic since the grammar file's options
// are TS-specific.
func mapToGrammarSpec(parsed map[string]any, ref map[jsonic.FuncRef]any) *jsonic.GrammarSpec {
	gs := &jsonic.GrammarSpec{
		Ref: ref,
	}

	ruleMap, _ := parsed["rule"].(map[string]any)
	if ruleMap == nil {
		return gs
	}

	gs.Rule = make(map[string]*jsonic.GrammarRuleSpec, len(ruleMap))
	for name, rDef := range ruleMap {
		rd, ok := rDef.(map[string]any)
		if !ok {
			continue
		}
		grs := &jsonic.GrammarRuleSpec{}
		if openDef, ok := rd["open"]; ok {
			grs.Open = convertAlts(openDef)
		}
		if closeDef, ok := rd["close"]; ok {
			grs.Close = convertAlts(closeDef)
		}
		gs.Rule[name] = grs
	}

	return gs
}

func convertAlts(def any) any {
	switch v := def.(type) {
	case []any:
		return convertAltList(v)
	case map[string]any:
		result := &jsonic.GrammarAltListSpec{}
		if alts, ok := v["alts"].([]any); ok {
			result.Alts = convertAltList(alts)
		}
		if inj, ok := v["inject"].(map[string]any); ok {
			result.Inject = &jsonic.GrammarInjectSpec{}
			if app, ok := inj["append"].(bool); ok {
				result.Inject.Append = app
			}
		}
		return result
	}
	return nil
}

func convertAltList(alts []any) []*jsonic.GrammarAltSpec {
	result := make([]*jsonic.GrammarAltSpec, 0, len(alts))
	for _, a := range alts {
		if am, ok := a.(map[string]any); ok {
			result = append(result, convertAlt(am))
		}
	}
	return result
}

func convertAlt(m map[string]any) *jsonic.GrammarAltSpec {
	ga := &jsonic.GrammarAltSpec{}

	if s, ok := m["s"]; ok {
		switch sv := s.(type) {
		case string:
			ga.S = sv
		case []any:
			strs := make([]string, len(sv))
			for i, v := range sv {
				strs[i], _ = v.(string)
			}
			ga.S = strs
		}
	}
	if b, ok := m["b"]; ok {
		ga.B = b
	}
	if p, ok := m["p"].(string); ok {
		ga.P = p
	}
	if r, ok := m["r"].(string); ok {
		ga.R = r
	}
	if a, ok := m["a"].(string); ok {
		ga.A = a
	}
	if c, ok := m["c"]; ok {
		ga.C = c
	}
	if e, ok := m["e"].(string); ok {
		ga.E = e
	}
	if u, ok := m["u"].(map[string]any); ok {
		ga.U = u
	}

	return ga
}
