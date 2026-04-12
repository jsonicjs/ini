/* Copyright (c) 2021-2025 Richard Rodger, MIT License */

package ini

import (
	"strings"

	jsonic "github.com/jsonicjs/jsonic/go"
)

type actionFunc = func(r *jsonic.Rule, ctx *jsonic.Context)
type condFunc = func(r *jsonic.Rule, ctx *jsonic.Context) bool

type parsedRule struct {
	open  []*jsonic.AltSpec
	close []*jsonic.AltSpec
}

// parseGrammarRules parses the embedded jsonic grammar text and builds
// rule specifications from the declarative definition.
func parseGrammarRules(
	tokenMap map[string]jsonic.Tin,
	actions map[string]actionFunc,
	conds map[string]condFunc,
) map[string]*parsedRule {
	parser := jsonic.Make()
	grammarDef, err := parser.Parse(grammarText)
	if err != nil {
		panic("failed to parse ini grammar: " + err.Error())
	}

	gd, _ := grammarDef.(map[string]any)
	rules, _ := gd["rule"].(map[string]any)

	result := make(map[string]*parsedRule)
	for name, ruleDef := range rules {
		rd, _ := ruleDef.(map[string]any)
		pr := &parsedRule{}

		if openDef, ok := rd["open"]; ok {
			pr.open = buildAltSpecs(openDef, tokenMap, actions, conds)
		}
		if closeDef, ok := rd["close"]; ok {
			pr.close = buildAltSpecs(closeDef, tokenMap, actions, conds)
		}

		result[name] = pr
	}

	return result
}

// buildAltSpecs handles both array format and {alts, inject} object format.
func buildAltSpecs(
	def any,
	tokenMap map[string]jsonic.Tin,
	actions map[string]actionFunc,
	conds map[string]condFunc,
) []*jsonic.AltSpec {
	var altDefs []any
	switch v := def.(type) {
	case []any:
		altDefs = v
	case map[string]any:
		// Object with alts/inject (e.g. map rule open).
		if alts, ok := v["alts"].([]any); ok {
			altDefs = alts
		}
	}

	var result []*jsonic.AltSpec
	for _, ad := range altDefs {
		if altMap, ok := ad.(map[string]any); ok {
			result = append(result, buildAltSpec(altMap, tokenMap, actions, conds))
		}
	}
	return result
}

func buildAltSpec(
	altDef map[string]any,
	tokenMap map[string]jsonic.Tin,
	actions map[string]actionFunc,
	conds map[string]condFunc,
) *jsonic.AltSpec {
	alt := &jsonic.AltSpec{}

	if s, ok := altDef["s"]; ok {
		alt.S = resolveTokenSeq(s, tokenMap)
	}
	if p, ok := altDef["p"].(string); ok {
		alt.P = p
	}
	if r, ok := altDef["r"].(string); ok {
		alt.R = r
	}
	if b, ok := altDef["b"]; ok {
		alt.B = toInt(b)
	}
	if a, ok := altDef["a"].(string); ok {
		if fn, exists := actions[a]; exists {
			alt.A = fn
		}
	}
	if c, ok := altDef["c"].(string); ok {
		if fn, exists := conds[c]; exists {
			alt.C = fn
		}
	}
	if u, ok := altDef["u"].(map[string]any); ok {
		alt.U = u
	}

	return alt
}

// resolveTokenSeq converts the grammar s field to [][]jsonic.Tin.
//
// String "s": "#OS" → [][]Tin{{OS}}
// String "s": "#HK #ST #VL" → [][]Tin{{HK, ST, VL}}  (alternatives at position 0)
// Array  "s": ["#HK #ST #VL", "#EQ"] → [][]Tin{{HK, ST, VL}, {EQ}}
func resolveTokenSeq(s any, tokenMap map[string]jsonic.Tin) [][]jsonic.Tin {
	switch v := s.(type) {
	case string:
		parts := strings.Fields(v)
		var tins []jsonic.Tin
		for _, p := range parts {
			if tin, ok := tokenMap[p]; ok {
				tins = append(tins, tin)
			}
		}
		if len(tins) == 0 {
			return nil
		}
		return [][]jsonic.Tin{tins}
	case []any:
		var result [][]jsonic.Tin
		for _, elem := range v {
			str, ok := elem.(string)
			if !ok {
				continue
			}
			parts := strings.Fields(str)
			var tins []jsonic.Tin
			for _, p := range parts {
				if tin, ok := tokenMap[p]; ok {
					tins = append(tins, tin)
				}
			}
			result = append(result, tins)
		}
		return result
	}
	return nil
}

func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

// insertAlt inserts an AltSpec at the given index in the slice.
func insertAlt(alts []*jsonic.AltSpec, idx int, alt *jsonic.AltSpec) []*jsonic.AltSpec {
	if idx >= len(alts) {
		return append(alts, alt)
	}
	alts = append(alts, nil)
	copy(alts[idx+1:], alts[idx:])
	alts[idx] = alt
	return alts
}

// wrapAction creates a new action that calls orig (if non-nil) then extra.
func wrapAction(orig actionFunc, extra actionFunc) actionFunc {
	if orig == nil {
		return extra
	}
	return func(r *jsonic.Rule, ctx *jsonic.Context) {
		orig(r, ctx)
		extra(r, ctx)
	}
}
