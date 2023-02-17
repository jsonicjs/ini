"use strict";
/* Copyright (c) 2021-2023 Richard Rodger and other contributors, MIT License */
Object.defineProperty(exports, "__esModule", { value: true });
const jsonic_next_1 = require("@jsonic/jsonic-next");
const ini_1 = require("../ini");
const j = jsonic_next_1.Jsonic.make().use(ini_1.Ini);
describe('ini', () => {
    test('happy', () => {
        expect(j('a=1')).toEqual({ a: "1" });
        expect(j('[A]')).toEqual({ A: {} });
        expect(j(`[A.B]\nc='2'`)).toEqual({ A: { B: { c: 2 } } });
        expect(j('a[]=1\na[]=2')).toEqual({ a: ['1', '2'] });
        expect(j(';X\n#Y\na=1;2')).toEqual({ a: '1' });
    });
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
            .toEqual({
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
        });
    });
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
#FIX j = "{ o: "p", a: { av: "a val", b: { c: { e: "this [value]" } } } }"
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
            .toEqual({
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
                "j": "\"{ o: \"p\", a: { av: \"a val\", b: { c: { e: \"this [value]\" } } } }\"",
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
                    "nocomment": "this; this is not a comment",
                    "noHashComment": "this# this is not a comment",
                },
                "x.y.z": "xyz",
            },
            "zr": [
                "deedee",
            ],
        });
    });
});
//# sourceMappingURL=ini.test.js.map