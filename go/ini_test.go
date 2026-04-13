/* Copyright (c) 2021-2025 Richard Rodger, MIT License */

package ini

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// assert is a test helper that checks deep equality.
func assert(t *testing.T, name string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n  got:  %#v\n  want: %#v", name, got, want)
	}
}

func TestDependencyVersions(t *testing.T) {
	modData, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	sumData, err := os.ReadFile("go.sum")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(modData) + "\n" + string(sumData)

	// Check jsonic version is 0.1.13 (may be a pseudo-version like v0.1.13-0.xxx)
	if !strings.Contains(combined, "github.com/jsonicjs/jsonic/go v0.1.13") {
		t.Errorf("expected jsonic version v0.1.13, not found in go.mod or go.sum")
	}

	// Check hoover version is 0.1.2 (transitive dep, appears in go.sum)
	if !strings.Contains(combined, "github.com/jsonicjs/hoover/go v0.1.2") {
		t.Errorf("expected hoover version v0.1.2, not found in go.mod or go.sum")
	}
}

func TestHappy(t *testing.T) {
	j := MakeJsonic()

	r, err := j.Parse("a=1")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "simple", r, map[string]any{"a": "1"})

	r, err = j.Parse("[A]")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "section", r, map[string]any{"A": map[string]any{}})

	r, err = j.Parse("a=\nb=")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "empty-values", r, map[string]any{"a": "", "b": ""})
}

func TestInlineCommentsOff(t *testing.T) {
	// Default: inline comments are off. ; and # mid-value are literal.
	result, err := Parse("a = hello ; world")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "semicolon-literal", result, map[string]any{"a": "hello ; world"})

	result, err = Parse("a = hello # world")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "hash-literal", result, map[string]any{"a": "hello # world"})

	result, err = Parse("a = x;y;z")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "multi-semi", result, map[string]any{"a": "x;y;z"})
}

func TestLineComments(t *testing.T) {
	// Line-start comments always work.
	result, err := Parse("; comment\na = 1")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "semi-comment", result, map[string]any{"a": "1"})

	result, err = Parse("# comment\na = 1")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "hash-comment", result, map[string]any{"a": "1"})
}

func TestInlineActive(t *testing.T) {
	result, err := Parse("a = hello ; comment", IniOptions{
		Comment: &CommentOptions{
			Inline: &InlineCommentOptions{Active: boolPtr(true)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "semi-inline", result, map[string]any{"a": "hello"})

	result, err = Parse("a = hello # comment", IniOptions{
		Comment: &CommentOptions{
			Inline: &InlineCommentOptions{Active: boolPtr(true)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "hash-inline", result, map[string]any{"a": "hello"})
}

func TestInlineCustomChars(t *testing.T) {
	result, err := Parse("a = hello ; comment\nb = hello # not a comment", IniOptions{
		Comment: &CommentOptions{
			Inline: &InlineCommentOptions{
				Active: boolPtr(true),
				Chars:  []string{";"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "custom-chars", result, map[string]any{
		"a": "hello",
		"b": "hello # not a comment",
	})
}

func TestInlineBackslashEscape(t *testing.T) {
	result, err := Parse("a = hello\\; world", IniOptions{
		Comment: &CommentOptions{
			Inline: &InlineCommentOptions{
				Active: boolPtr(true),
				Escape: &InlineEscapeOptions{Backslash: boolPtr(true)},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "backslash-semi", result, map[string]any{"a": "hello; world"})

	result, err = Parse("a = hello\\# world", IniOptions{
		Comment: &CommentOptions{
			Inline: &InlineCommentOptions{
				Active: boolPtr(true),
				Escape: &InlineEscapeOptions{Backslash: boolPtr(true)},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "backslash-hash", result, map[string]any{"a": "hello# world"})
}

func TestInlineWhitespacePrefix(t *testing.T) {
	result, err := Parse("a = x;y;z", IniOptions{
		Comment: &CommentOptions{
			Inline: &InlineCommentOptions{
				Active: boolPtr(true),
				Escape: &InlineEscapeOptions{Whitespace: boolPtr(true)},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "no-ws-literal", result, map[string]any{"a": "x;y;z"})

	result, err = Parse("a = hello ;comment", IniOptions{
		Comment: &CommentOptions{
			Inline: &InlineCommentOptions{
				Active: boolPtr(true),
				Escape: &InlineEscapeOptions{Whitespace: boolPtr(true)},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "ws-comment", result, map[string]any{"a": "hello"})
}

func TestSections(t *testing.T) {
	result, err := Parse("[d]\ne = 2")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "simple-section", result, map[string]any{
		"d": map[string]any{"e": "2"},
	})
}

func TestNestedSections(t *testing.T) {
	result, err := Parse("[h.i]\nj = 3")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "nested", result, map[string]any{
		"h": map[string]any{
			"i": map[string]any{"j": "3"},
		},
	})
}

func TestSectionDuplicateMerge(t *testing.T) {
	result, err := Parse("[a]\nx=1\ny=2\n[a]\nz=3")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "merge", result, map[string]any{
		"a": map[string]any{"x": "1", "y": "2", "z": "3"},
	})
}

func TestSectionDuplicateOverride(t *testing.T) {
	result, err := Parse("[a]\nx=1\ny=2\n[a]\nz=3", IniOptions{
		Section: &SectionOptions{Duplicate: "override"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "override", result, map[string]any{
		"a": map[string]any{"z": "3"},
	})
}

func TestSectionDuplicateError(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for duplicate section")
		}
	}()
	Parse("[a]\nx=1\n[a]\ny=2", IniOptions{
		Section: &SectionOptions{Duplicate: "error"},
	})
}

func TestKeyByItself(t *testing.T) {
	// Bare key (without =) means key=true. Works after a key=value pair
	// (matching TS behavior where pair.close routes bare keys back to pair).
	result, err := Parse("a=1\nmykey")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "key-true", result, map[string]any{"a": "1", "mykey": true})
}

func TestArraySyntax(t *testing.T) {
	result, err := Parse("a[]=1\na[]=2")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "array", result, map[string]any{"a": []any{"1", "2"}})
}

func TestMultilineContinuation(t *testing.T) {
	result, err := Parse("a = hello \\\nworld", IniOptions{
		Multiline: &MultilineOptions{},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "backslash-cont", result, map[string]any{"a": "hello world"})

	// Multiple continuations.
	result, err = Parse("a = one \\\ntwo \\\nthree", IniOptions{
		Multiline: &MultilineOptions{},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "multi-cont", result, map[string]any{"a": "one two three"})
}

func TestMultilineIndent(t *testing.T) {
	noBackslash := ""
	result, err := Parse("a = hello\n    world", IniOptions{
		Multiline: &MultilineOptions{
			Indent:       boolPtr(true),
			Continuation: &noBackslash,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "indent-cont", result, map[string]any{"a": "hello world"})
}

func TestMultilineWithInlineComments(t *testing.T) {
	result, err := Parse("a = hello \\\nworld ;comment\nb = 2", IniOptions{
		Multiline: &MultilineOptions{},
		Comment: &CommentOptions{
			Inline: &InlineCommentOptions{Active: boolPtr(true)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "multiline-inline", result, map[string]any{
		"a": "hello world",
		"b": "2",
	})
}

func TestQuotedValues(t *testing.T) {
	result, err := Parse(`a = "hello world"`)
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "double-quoted", result, map[string]any{"a": "hello world"})

	result, err = Parse("a = 'hello world'")
	if err != nil {
		t.Fatal(err)
	}
	// Single-quoted values attempt JSON parse.
	assert(t, "single-quoted", result, map[string]any{"a": "hello world"})
}

func TestEmptyInput(t *testing.T) {
	result, err := Parse("")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "empty", result, map[string]any{})
}

func TestBooleanValues(t *testing.T) {
	result, err := Parse("a = true\nb = false")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "booleans", result, map[string]any{"a": true, "b": false})
}

func TestNullValue(t *testing.T) {
	result, err := Parse("a = null")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "null", result, map[string]any{"a": nil})
}

func TestMultiplePairs(t *testing.T) {
	result, err := Parse("a = 1\nb = x\nc = y y")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "multi-pairs", result, map[string]any{
		"a": "1",
		"b": "x",
		"c": "y y",
	})
}

func TestMixedSectionsAndPairs(t *testing.T) {
	result, err := Parse("x = 0\n[s]\na = 1\nb = 2")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "mixed", result, map[string]any{
		"x": "0",
		"s": map[string]any{"a": "1", "b": "2"},
	})
}

func TestUsePlugin(t *testing.T) {
	// Verify the plugin interface works directly.
	j := MakeJsonic()
	result, err := j.Parse("a=1\nb=2")
	if err != nil {
		t.Fatal(err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	assert(t, "plugin", m, map[string]any{"a": "1", "b": "2"})
}

func TestEqualsInValue(t *testing.T) {
	result, err := Parse("u = v = 5")
	if err != nil {
		t.Fatal(err)
	}
	assert(t, "eq-in-value", result, map[string]any{"u": "v = 5"})
}
