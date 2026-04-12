/* Copyright (c) 2021-2025 Richard Rodger, MIT License */

package ini

import (
	"fmt"
	"strings"

	jsonic "github.com/jsonicjs/jsonic/go"
)

// lexMode tracks which kind of token the custom matchers should produce.
// Set by grammar rule callbacks, read by matchers.
type lexMode int

const (
	modeKey  lexMode = iota // Scanning a key (until = or newline)
	modeVal                 // Scanning a value (until newline or comment)
	modeDive                // Scanning section path part (until ] or .)
	modeNone                // Don't produce custom tokens
)

// Ini is a jsonic plugin that adds INI parsing support.
func Ini(j *jsonic.Jsonic, pluginOpts map[string]any) {
	opts := mapToResolved(pluginOpts)

	// Closure state for context-dependent lexing.
	mode := modeKey

	cfg := j.Config()

	// Disable JSON structure tokens except [ and ].
	delete(cfg.FixedTokens, "{")
	delete(cfg.FixedTokens, "}")
	delete(cfg.FixedTokens, ":")
	cfg.SortFixedTokens()

	// Register custom fixed tokens.
	EQ := j.Token("#EQ", "=")
	DOT := j.Token("#DOT", ".")
	cfg.SortFixedTokens()

	// Register custom token types for INI-specific blocks.
	HK := j.Token("#HK") // Hoover Key
	HV := j.Token("#HV") // Hoover Value
	DK := j.Token("#DK") // Dive Key (section path part)

	// Standard tokens.
	ZZ := j.Token("#ZZ")
	OS := j.Token("#OS") // [
	CS := j.Token("#CS") // ]
	ST := j.Token("#ST") // String
	VL := j.Token("#VL") // Value
	CL := j.Token("#CL") // Colon (disabled but needed for grammar file token resolution)
	_ = DOT              // used by grammar
	_ = VL               // used by grammar
	_ = CL               // used by grammar

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
	// All at priority < 2e6 so they run before built-in matchers.
	// They must return nil for chars they don't handle (spaces, newlines,
	// fixed tokens, etc.) so the built-in matchers can process them.

	// Key matcher: reads a key token (until =, newline, [, ], or EOF).
	j.AddMatcher("inikey", 100000, func(lex *jsonic.Lex, rule *jsonic.Rule) *jsonic.Token {
		if mode != modeKey {
			return nil
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
		// Pass [ unless it's part of key[] array syntax.
		// At start of key, [ is a section opener.
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
	// Handles leading whitespace, multiline continuation, and inline comments.
	// Also handles the empty-value case (newline/EOF immediately after =).
	j.AddMatcher("inival", 100001, func(lex *jsonic.Lex, rule *jsonic.Rule) *jsonic.Token {
		if mode != modeVal {
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
	j.AddMatcher("inidive", 100002, func(lex *jsonic.Lex, rule *jsonic.Rule) *jsonic.Token {
		if mode != modeDive {
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
	// Rules ini, table, dive, map, pair are loaded from ini-grammar.jsonic.
	// The val rule is defined in Go code (needs custom logic not in the grammar file).

	var declaredSections map[string]bool

	// Token map for resolving grammar file token references.
	tokenMap := map[string]jsonic.Tin{
		"#HK": HK, "#HV": HV, "#DK": DK,
		"#EQ": EQ, "#DOT": DOT,
		"#OS": OS, "#CS": CS,
		"#ST": ST, "#VL": VL, "#CL": CL,
		"#ZZ": ZZ,
	}

	// Action function refs (matching @ names in the grammar file).
	// Go-specific mode switching is included in these functions.
	actions := map[string]actionFunc{
		"@table-close-dive": func(r *jsonic.Rule, ctx *jsonic.Context) {
			if r.Child != nil && r.Child != jsonic.NoRule {
				if dive, ok := r.Child.U["dive"].([]string); ok {
					r.U["dive"] = dive
				}
			}
			mode = modeKey
		},

		"@dive-push": func(r *jsonic.Rule, ctx *jsonic.Context) {
			dive := getDive(r.Parent)
			val, _ := r.O0.Val.(string)
			dive = append(dive, val)
			r.U["dive"] = dive
			if r.Parent != nil && r.Parent != jsonic.NoRule {
				r.Parent.U["dive"] = dive
			}
		},

		"@pair-key-eq": func(r *jsonic.Rule, ctx *jsonic.Context) {
			mode = modeVal
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
		},

		"@pair-key-bool": func(r *jsonic.Rule, ctx *jsonic.Context) {
			key := tokenString(r.O0)
			if key != "" {
				if nodeMap, ok := r.Parent.Node.(map[string]any); ok {
					nodeMap[key] = true
				}
			}
		},

		"@pair-close-err": func(r *jsonic.Rule, ctx *jsonic.Context) {
			// Error handler: not used in Go (CL token is disabled).
		},

		"@val-empty": func(r *jsonic.Rule, ctx *jsonic.Context) {
			r.Node = ""
		},
	}

	// Condition function refs.
	conds := map[string]condFunc{
		"@is-table-parent": func(r *jsonic.Rule, ctx *jsonic.Context) bool {
			return r.Parent != nil && r.Parent.Name == "table"
		},
		"@is-table-grandparent": func(r *jsonic.Rule, ctx *jsonic.Context) bool {
			return r.Parent != nil && r.Parent.Parent != nil &&
				r.Parent.Parent.Name == "table"
		},
	}

	// Parse and build grammar rules from the embedded jsonic file.
	grammarRules := parseGrammarRules(tokenMap, actions, conds)

	// Apply grammar rules with Go-specific state actions and mode switching.

	// Go-specific: bare KEY alts needed because Go's custom key matcher
	// produces HK tokens for bare keys (without =), unlike TS's Hoover.
	KEY := []jsonic.Tin{HK, ST}
	bareKeyToTable := &jsonic.AltSpec{S: [][]jsonic.Tin{KEY}, P: "table", B: 1}
	bareKeyToMap := &jsonic.AltSpec{S: [][]jsonic.Tin{KEY}, P: "map", B: 1}

	j.Rule("ini", func(rs *jsonic.RuleSpec) {
		rs.Clear()

		rs.AddBO(func(r *jsonic.Rule, ctx *jsonic.Context) {
			r.Node = make(map[string]any)
			declaredSections = make(map[string]bool)
			mode = modeKey
		})

		if gr, ok := grammarRules["ini"]; ok {
			rs.Open = gr.open
			rs.Close = gr.close
		}
		// Insert bare KEY alt after KEY+EQ alt (index 1) for Go custom matchers.
		rs.Open = insertAlt(rs.Open, 2, bareKeyToTable)
	})

	j.Rule("table", func(rs *jsonic.RuleSpec) {
		rs.Clear()

		rs.AddBO(func(r *jsonic.Rule, ctx *jsonic.Context) {
			r.Node = r.Parent.Node
			mode = modeKey

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
		})

		rs.AddBC(func(r *jsonic.Rule, ctx *jsonic.Context) {
			if childMap, ok := r.Child.Node.(map[string]any); ok {
				if nodeMap, ok := r.Node.(map[string]any); ok {
					for k, v := range childMap {
						nodeMap[k] = v
					}
				}
			}
		})

		if gr, ok := grammarRules["table"]; ok {
			rs.Open = gr.open
			rs.Close = gr.close
		}
		// Insert bare KEY alt after KEY+EQ alt (index 1) for Go custom matchers.
		rs.Open = insertAlt(rs.Open, 2, bareKeyToMap)

		// Go-specific: set mode=modeDive on the first open alt (OS → dive).
		if len(rs.Open) > 0 {
			rs.Open[0].A = wrapAction(rs.Open[0].A, func(r *jsonic.Rule, ctx *jsonic.Context) {
				mode = modeDive
			})
		}
		// Go-specific: set mode=modeKey on the first close alt (OS → table).
		if len(rs.Close) > 0 {
			rs.Close[0].A = wrapAction(rs.Close[0].A, func(r *jsonic.Rule, ctx *jsonic.Context) {
				mode = modeKey
			})
		}
	})

	j.Rule("dive", func(rs *jsonic.RuleSpec) {
		rs.Clear()

		if gr, ok := grammarRules["dive"]; ok {
			rs.Open = gr.open
			rs.Close = gr.close
		}

		// Go-specific: set mode=modeKey on dive close (CS).
		if len(rs.Close) > 0 {
			rs.Close[0].A = wrapAction(rs.Close[0].A, func(r *jsonic.Rule, ctx *jsonic.Context) {
				mode = modeKey
			})
		}
	})

	j.Rule("map", func(rs *jsonic.RuleSpec) {
		rs.Clear()

		if gr, ok := grammarRules["map"]; ok {
			rs.Open = gr.open
			rs.Close = gr.close
		}
	})

	j.Rule("pair", func(rs *jsonic.RuleSpec) {
		rs.Clear()

		if gr, ok := grammarRules["pair"]; ok {
			rs.Open = gr.open
			rs.Close = gr.close
		}

		rs.AddAC(func(r *jsonic.Rule, ctx *jsonic.Context) {
			mode = modeKey
		})
	})

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
			mode = modeKey

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
