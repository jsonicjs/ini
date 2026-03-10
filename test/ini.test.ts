/* Copyright (c) 2021-2025 Richard Rodger and other contributors, MIT License */

import { test, describe } from 'node:test'
import { expect } from '@hapi/code'

import { Jsonic } from 'jsonic'
import { Ini } from '../dist/ini'


const j = Jsonic.make().use(Ini)


describe('ini', () => {

  test('happy', () => {
    expect(j('a=1')).equal({ a: "1" })
    expect(j('[A]')).equal({ A: {} })
    expect(j(`[A.B]\nc='2'`)).equal({ A: { B: { c: 2 } } })
    expect(j('a[]=1\na[]=2')).equal({ a: ['1', '2'] })
    expect(j('a=\nb=')).equal({ a: '', b: '' })
    // Inline comments are off by default; ; and # mid-value are literal.
    expect(j(';X\n#Y\na=1;2\nb=2')).equal({ a: '1;2', b: '2' })
  })


  test('basic', () => {
    expect(j(`
; comment
a = 1
b = x
c = y y
c0 = true
" c1  c2 " = null
'[]'='[]'

[d]
e = 2
e0[]=q q
e0[]=w w
"[]"="[]"

[f]
# x:11
g = 'G'
# x:12


[h.i]
j = [3,4]
j0 = ]3,4[
k = false

[l.m.n.o]
p = "P"
q = {x:1}
u = v = 5
w = '{"y":{"z":6}}'
aa = 7

`))
      .equal({
        a: '1',
        b: 'x',
        c: 'y y',
        c0: true,
        ' c1  c2 ': null,
        '[]': [],
        d: {
          e: '2',
          e0: ['q q', 'w w'],
          '[]': '[]',
        },
        f: { g: 'G' },
        h: { i: { j: '[3,4]', j0: ']3,4[', k: false } },
        l: {
          m: {
            n: {
              o: {
                p: 'P',
                q: '{x:1}',
                u: 'v = 5',
                w: { y: { z: 6 } },
                aa: '7'
              },
            }
          }
        }
      })
  })

  // NOTE: Copyright (c) Isaac Z. Schlueter and Contributors, ISC License
  test('ini-module-test', () => {
    expect(j(`
o = p

a with spaces   =     b  c

; wrap in quotes to JSON-decode and preserve spaces
" xa  n          p " = "\\"\\r\\nyoyoyo\\r\\r\\n"

; wrap in quotes to get a key with a bracket, not a section.
"[disturbing]" = hey you never know

; Test single quotes
s = 'something'

; Test mixing quotes

s1 = "something'

; Test double quotes
s2 = "something else"

; Test blank value
s3 =

; Test value with only spaces
s4 =

; Test quoted value with only spaces
s5 = '   '

; Test quoted value with leading and trailing spaces
s6 = ' a '

; Test no equal sign
s7

; Test bool(true)
true = true

; Test bool(false)
false = false

; Test null
null = null

; Test undefined
undefined = undefined

; Test arrays
zr[] = deedee
ar[] = one
ar[] = three
; This should be included in the array
ar   = this is included

; Test resetting of a value (and not turn it into an array)
br = cold
br = warm

eq = "eq=eq"

; a section
[a]
av = a val
e = { o: p, a: { av: a val, b: { c: { e: 'this [value]' } } } }
j = "{ o: \\"p\\", a: { av: \\"a val\\", b: { c: { e: \\"this [value]\\" } } } }"
"[]" = a square?

; Nested array
cr[] = four
cr[] = eight

; nested child without middle parent
; should create otherwise-empty a.b
[a.b.c]
e = 1
j = 2

; dots in the section name should be literally interpreted
[x\\.y\\.z]
x.y.z = xyz

[x\\.y\\.z.a\\.b\\.c]
a.b.c = abc

; this next one is not a comment!  it's escaped!
nocomment = this\\; this is not a comment

# Support the use of the number sign (#) as an alternative to the semicolon for indicating comments.
# http://en.wikipedia.org/wiki/INI_file#Comments

# this next one is not a comment!  it's escaped!
noHashComment = this\\# this is not a comment`))
      .equal({
        " xa  n          p ": "\"\r\nyoyoyo\r\r\n",
        "[disturbing]": "hey you never know",
        "a": {
          "[]": "a square?",
          "av": "a val",
          "b": {
            "c": {
              "e": "1",
              "j": "2",
            },
          },
          "cr": [
            "four",
            "eight",
          ],
          "e": "{ o: p, a: { av: a val, b: { c: { e: 'this [value]' } } } }",
          "j": "{ o: \"p\", a: { av: \"a val\", b: { c: { e: \"this [value]\" } } } }",
        },
        "a with spaces": "b  c",
        "ar": [
          "one",
          "three",
          "this is included",
        ],
        "br": "warm",
        "eq": "eq=eq",
        "false": false,
        "null": null,
        "o": "p",
        "s": "something",
        "s1": "\"something'",
        "s2": "something else",
        "s3": "",
        "s4": "",
        "s5": "   ",
        "s6": " a ",
        "s7": true,
        "true": true,
        "undefined": "undefined",
        "x.y.z": {
          "a.b.c": {
            "a.b.c": "abc",
            "nocomment": "this\\; this is not a comment",
            "noHashComment": "this\\# this is not a comment",
          },
          "x.y.z": "xyz",
        },
        "zr": [
          "deedee",
        ],
      })
  })


})


describe('multiline', () => {

  test('backslash-continuation', () => {
    const jm = Jsonic.make().use(Ini, { multiline: true })

    // Basic continuation with \<LF>
    expect(jm('a = hello \\\nworld')).equal({ a: 'hello world' })

    // Continuation with leading whitespace on next line (consumed)
    expect(jm('a = hello \\\n    world')).equal({ a: 'hello world' })

    // Multiple continuations
    expect(jm('a = one \\\ntwo \\\nthree')).equal({ a: 'one two three' })

    // No continuation: normal newline ends value
    expect(jm('a = hello\nb = world')).equal({ a: 'hello', b: 'world' })

    // Continuation with \<CR><LF>
    expect(jm('a = hello \\\r\nworld')).equal({ a: 'hello world' })

    // Escaped backslash before newline is NOT continuation
    expect(jm('a = path\\\\\nb = next')).equal({ a: 'path\\', b: 'next' })

    // Continuation in a section
    expect(jm('[s]\na = hello \\\n    world')).equal({ s: { a: 'hello world' } })

    // Empty value with continuation
    expect(jm('a = \\\nworld')).equal({ a: 'world' })

    // Inline comments off by default: ; is literal in value
    expect(jm('a = hello \\\nworld ;not-a-comment\nb = 2'))
      .equal({ a: 'hello world ;not-a-comment', b: '2' })
  })

  test('indent-continuation', () => {
    const ji = Jsonic.make().use(Ini, { multiline: { indent: true, continuation: false } })

    // Indented line continues previous value
    expect(ji('a = hello\n    world')).equal({ a: 'hello world' })

    // Multiple indent continuations
    expect(ji('a = line1\n  line2\n  line3')).equal({ a: 'line1 line2 line3' })

    // Non-indented line is a new key
    expect(ji('a = hello\nb = world')).equal({ a: 'hello', b: 'world' })

    // Tab indent
    expect(ji('a = hello\n\tworld')).equal({ a: 'hello world' })

    // Indent continuation in section
    expect(ji('[s]\na = hello\n    world'))
      .equal({ s: { a: 'hello world' } })
  })

  test('multiline-with-boolean-option', () => {
    // multiline: true enables defaults (backslash continuation, no indent)
    const jm = Jsonic.make().use(Ini, { multiline: true })
    expect(jm('a = hello \\\nworld')).equal({ a: 'hello world' })
  })

  test('multiline-both-modes', () => {
    // Both continuation char and indent enabled
    const jb = Jsonic.make().use(Ini, {
      multiline: { continuation: '\\', indent: true }
    })

    // Backslash continuation works
    expect(jb('a = hello \\\nworld')).equal({ a: 'hello world' })

    // Indent continuation also works
    expect(jb('a = hello\n    world')).equal({ a: 'hello world' })
  })

  test('multiline-escapes', () => {
    // Multiline with inline comments active and backslash escaping
    const jm = Jsonic.make().use(Ini, {
      multiline: true,
      comment: { inline: { active: true, escape: { backslash: true } } },
    })

    // Escaped comment chars still work with continuation
    expect(jm('a = one\\; two \\\nthree'))
      .equal({ a: 'one; two three' })

    // Escaped hash
    expect(jm('a = one\\# two \\\nthree'))
      .equal({ a: 'one# two three' })
  })

  test('multiline-no-inline-comments', () => {
    // Multiline without inline comments: ; and # are literal
    const jm = Jsonic.make().use(Ini, { multiline: true })

    expect(jm('a = one; two \\\nthree'))
      .equal({ a: 'one; two three' })

    expect(jm('a = one# two \\\nthree'))
      .equal({ a: 'one# two three' })
  })
})


describe('section-duplicate', () => {

  test('merge-default', () => {
    const j = Jsonic.make().use(Ini)

    // Default: merge keys from duplicate sections
    expect(j('[a]\nx=1\ny=2\n[a]\nz=3'))
      .equal({ a: { x: '1', y: '2', z: '3' } })

    // Duplicate key: last value wins
    expect(j('[a]\nx=1\n[a]\nx=2'))
      .equal({ a: { x: '2' } })

    // Nested duplicate sections merge
    expect(j('[a.b]\nx=1\n[a.b]\ny=2'))
      .equal({ a: { b: { x: '1', y: '2' } } })

    // Intermediate path preserved when merging
    expect(j('[a.b]\nx=1\n[a]\ny=2'))
      .equal({ a: { b: { x: '1' }, y: '2' } })
  })

  test('merge-explicit', () => {
    const jm = Jsonic.make().use(Ini, { section: { duplicate: 'merge' } })

    expect(jm('[a]\nx=1\n[a]\ny=2'))
      .equal({ a: { x: '1', y: '2' } })
  })

  test('override', () => {
    const jo = Jsonic.make().use(Ini, { section: { duplicate: 'override' } })

    // Second occurrence replaces first
    expect(jo('[a]\nx=1\ny=2\n[a]\nz=3'))
      .equal({ a: { z: '3' } })

    // First occurrence works normally
    expect(jo('[a]\nx=1'))
      .equal({ a: { x: '1' } })

    // Override clears subsections too
    expect(jo('[a.b]\nx=1\n[a]\ny=2\n[a]\nz=3'))
      .equal({ a: { z: '3' } })

    // Non-duplicate sections unaffected
    expect(jo('[a]\nx=1\n[b]\ny=2'))
      .equal({ a: { x: '1' }, b: { y: '2' } })

    // Nested override
    expect(jo('[a.b]\nx=1\n[a.b]\ny=2'))
      .equal({ a: { b: { y: '2' } } })
  })

  test('error', () => {
    const je = Jsonic.make().use(Ini, { section: { duplicate: 'error' } })

    // Single section: no error
    expect(je('[a]\nx=1')).equal({ a: { x: '1' } })

    // Multiple distinct sections: no error
    expect(je('[a]\nx=1\n[b]\ny=2'))
      .equal({ a: { x: '1' }, b: { y: '2' } })

    // Duplicate section: throws
    expect(() => je('[a]\nx=1\n[a]\ny=2'))
      .to.throw(/Duplicate section/)

    // Duplicate nested section: throws
    expect(() => je('[a.b]\nx=1\n[a.b]\ny=2'))
      .to.throw(/Duplicate section/)

    // Intermediate path is NOT a declared section
    // [a.b] creates intermediate [a] but does not declare it
    expect(je('[a.b]\nx=1\n[a]\ny=2'))
      .equal({ a: { b: { x: '1' }, y: '2' } })
  })
})


describe('inline-comment', () => {

  test('off-by-default', () => {
    // Default: inline comments are off. ; and # mid-value are literal.
    const j = Jsonic.make().use(Ini)

    expect(j('a = hello ; world')).equal({ a: 'hello ; world' })
    expect(j('a = hello # world')).equal({ a: 'hello # world' })
    expect(j('a = x;y;z')).equal({ a: 'x;y;z' })

    // Line-start comments still work
    expect(j('; comment\na = 1')).equal({ a: '1' })
    expect(j('# comment\na = 1')).equal({ a: '1' })
  })

  test('active-basic', () => {
    // Inline comments active with defaults (chars: ['#', ';'])
    const j = Jsonic.make().use(Ini, {
      comment: { inline: { active: true } },
    })

    expect(j('a = hello ; comment')).equal({ a: 'hello' })
    expect(j('a = hello # comment')).equal({ a: 'hello' })
    expect(j('a = x;y')).equal({ a: 'x' })
    expect(j('a = value\nb = other')).equal({ a: 'value', b: 'other' })
  })

  test('custom-chars', () => {
    // Only ; is an inline comment char, not #
    const j = Jsonic.make().use(Ini, {
      comment: { inline: { active: true, chars: [';'] } },
    })

    expect(j('a = hello ; comment')).equal({ a: 'hello' })
    expect(j('a = hello # not a comment')).equal({ a: 'hello # not a comment' })
  })

  test('backslash-escape', () => {
    // Backslash escaping enabled (default when active)
    const j = Jsonic.make().use(Ini, {
      comment: { inline: { active: true, escape: { backslash: true } } },
    })

    expect(j('a = hello\\; world')).equal({ a: 'hello; world' })
    expect(j('a = hello\\# world')).equal({ a: 'hello# world' })
    expect(j('a = x\\;y ; comment')).equal({ a: 'x;y' })
  })

  test('backslash-escape-disabled', () => {
    // Backslash escaping explicitly disabled: \; keeps both chars but
    // the escapeChar still prevents ; from terminating. The difference
    // is that backslash is preserved in the output rather than consumed.
    const j = Jsonic.make().use(Ini, {
      comment: { inline: { active: true, escape: { backslash: false } } },
    })

    // \; → \; (backslash preserved, ; did not terminate)
    expect(j('a = hello\\; world')).equal({ a: 'hello\\; world' })

    // Unescaped ; still terminates
    expect(j('a = hello ; comment')).equal({ a: 'hello' })
  })

  test('whitespace-prefix', () => {
    // Whitespace-prefix mode: only treat as comment if preceded by whitespace
    const j = Jsonic.make().use(Ini, {
      comment: { inline: { active: true, escape: { whitespace: true } } },
    })

    // No whitespace before ;  →  literal
    expect(j('a = x;y;z')).equal({ a: 'x;y;z' })

    // Whitespace before ;  →  inline comment
    expect(j('a = hello ;comment')).equal({ a: 'hello' })
    expect(j('a = hello\t;comment')).equal({ a: 'hello' })

    // Same for #
    expect(j('a = x#y')).equal({ a: 'x#y' })
    expect(j('a = hello #comment')).equal({ a: 'hello' })
  })

  test('whitespace-prefix-with-backslash', () => {
    // Both whitespace and backslash escaping
    const j = Jsonic.make().use(Ini, {
      comment: {
        inline: {
          active: true,
          escape: { whitespace: true, backslash: true },
        },
      },
    })

    // No whitespace: literal
    expect(j('a = x;y')).equal({ a: 'x;y' })

    // Whitespace present: comment
    expect(j('a = hello ;comment')).equal({ a: 'hello' })

    // Backslash escape overrides whitespace: literal
    expect(j('a = hello \\;not-a-comment')).equal({ a: 'hello ;not-a-comment' })
  })

  test('with-multiline', () => {
    // Inline comments active + multiline continuation
    const j = Jsonic.make().use(Ini, {
      multiline: true,
      comment: { inline: { active: true } },
    })

    // Comment terminates continued value
    expect(j('a = hello \\\nworld ;comment\nb = 2'))
      .equal({ a: 'hello world', b: '2' })

    // Escaped comment char in multiline value
    expect(j('a = hello\\; \\\nworld'))
      .equal({ a: 'hello; world' })
  })

  test('with-sections', () => {
    const j = Jsonic.make().use(Ini, {
      comment: { inline: { active: true } },
    })

    expect(j('[s]\na = val ; comment\nb = other'))
      .equal({ s: { a: 'val', b: 'other' } })
  })

  test('line-comments-always-work', () => {
    // Line-start comments work regardless of inline comment setting
    const jOff = Jsonic.make().use(Ini)
    const jOn = Jsonic.make().use(Ini, {
      comment: { inline: { active: true } },
    })

    const input = '; line comment\n# hash comment\na = 1'
    expect(jOff(input)).equal({ a: '1' })
    expect(jOn(input)).equal({ a: '1' })
  })
})


describe('number-lex', () => {

  // Enable number lexing via post-config so Jsonic parses numeric values as numbers
  function makeWithNumbers() {
    const jn = Jsonic.make().use(Ini)
    jn.options({ number: { lex: true } })
    return jn
  }

  test('integers', () => {
    const jn = makeWithNumbers()

    expect(jn('a=1')).equal({ a: 1 })
    expect(jn('a=0')).equal({ a: 0 })
    expect(jn('a=-3')).equal({ a: -3 })
    expect(jn('a=+2')).equal({ a: 2 })
    expect(jn('a=42\nb=99')).equal({ a: 42, b: 99 })
  })

  test('floats', () => {
    const jn = makeWithNumbers()

    expect(jn('a=2.5')).equal({ a: 2.5 })
    expect(jn('a=0.0')).equal({ a: 0 })
    expect(jn('a=-1.25')).equal({ a: -1.25 })
  })

  test('scientific-notation', () => {
    const jn = makeWithNumbers()

    expect(jn('a=1e10')).equal({ a: 1e10 })
  })

  test('hex', () => {
    const jn = makeWithNumbers()

    expect(jn('a=0xFF')).equal({ a: 255 })
  })

  test('mixed-types', () => {
    const jn = makeWithNumbers()

    // Numbers and strings coexist
    expect(jn('a=1\nb=hello\nc=2.5\nd=true'))
      .equal({ a: 1, b: 'hello', c: 2.5, d: true })

    // Non-numeric strings stay as strings
    expect(jn('a=1abc')).equal({ a: '1abc' })

    // Empty value stays as empty string
    expect(jn('a=\nb=1')).equal({ a: '', b: 1 })
  })

  test('in-sections', () => {
    const jn = makeWithNumbers()

    expect(jn('[s]\na=42\nb=text'))
      .equal({ s: { a: 42, b: 'text' } })

    expect(jn('[s]\na=1\n[t]\nb=2'))
      .equal({ s: { a: 1 }, t: { b: 2 } })
  })

  test('arrays', () => {
    const jn = makeWithNumbers()

    expect(jn('a[]=1\na[]=2\na[]=hello'))
      .equal({ a: [1, 2, 'hello'] })
  })

  test('default-numbers-are-strings', () => {
    // Without number.lex, all values are strings
    const j = Jsonic.make().use(Ini)

    expect(j('a=1')).equal({ a: '1' })
    expect(j('a=2.5')).equal({ a: '2.5' })
    expect(j('a=-3')).equal({ a: '-3' })
    expect(j('a=0xFF')).equal({ a: '0xFF' })
  })
})
