"use strict";
/* Copyright (c) 2021 Richard Rodger and other contributors, MIT License */
Object.defineProperty(exports, "__esModule", { value: true });
const jsonic_next_1 = require("@jsonic/jsonic-next");
const jsonc_1 = require("../jsonc");
const j = jsonic_next_1.Jsonic.make().use(jsonc_1.Jsonc);
describe('jsonc', () => {
    test('happy', () => {
        expect(j('{"a":1}')).toEqual({ a: 1 });
    });
    test('comments', () => {
        expect(j('// this is a comment')).toEqual(undefined);
        expect(j('// this is a comment\n')).toEqual(undefined);
        expect(j('/* this is a comment*/')).toEqual(undefined);
        expect(j('/* this is a \r\ncomment*/')).toEqual(undefined);
        expect(j('/* this is a \ncomment*/')).toEqual(undefined);
        expect(() => j('/* this is a')).toThrow('unterminated_comment');
        expect(() => j('/* this is a \ncomment')).toThrow('unterminated_comment');
        expect(() => j('/ ttt')).toThrow('unexpected');
    });
    test('strings', () => {
        expect(j('"test"')).toEqual('test');
        expect(j('"\\""')).toEqual('"');
        expect(j('"\\/"')).toEqual('/');
        expect(j('"\\b"')).toEqual('\b');
        expect(j('"\\f"')).toEqual('\f');
        expect(j('"\\n"')).toEqual('\n');
        expect(j('"\\r"')).toEqual('\r');
        expect(j('"\\t"')).toEqual('\t');
        expect(j('"\u88ff"')).toEqual('\u88ff');
        expect(j('"​\u2028"')).toEqual('​\u2028');
        expect(() => j('"\\v"')).toThrow('unexpected');
        expect(() => j('"test')).toThrow('unterminated_string');
        expect(() => j('"test\n"')).toThrow('unprintable');
        expect(() => j('"\t"')).toThrow('unprintable');
        expect(() => j('"\t "')).toThrow('unprintable');
        expect(() => j('"\0 "')).toThrow('unprintable');
    });
    test('numbers', () => {
        expect(j('0')).toEqual(0);
        expect(j('0.1')).toEqual(0.1);
        expect(j('-0.1')).toEqual(-0.1);
        expect(j('-1')).toEqual(-1);
        expect(j('1')).toEqual(1);
        expect(j('123456789')).toEqual(123456789);
        expect(j('10')).toEqual(10);
        expect(j('90')).toEqual(90);
        expect(j('90E+123')).toEqual(90E+123);
        expect(j('90e+123')).toEqual(90e+123);
        expect(j('90e-123')).toEqual(90e-123);
        expect(j('90E-123')).toEqual(90E-123);
        expect(j('90E123')).toEqual(90E123);
        expect(j('90e123')).toEqual(90e123);
        expect(j('01')).toEqual(1);
        expect(j('-01')).toEqual(-1);
        expect(() => j('-')).toThrow('unexpected');
        expect(() => j('.0')).toThrow('unexpected');
    });
    test('keywords', () => {
        expect(j('true')).toEqual(true);
        expect(j('false')).toEqual(false);
        expect(j('null')).toEqual(null);
        expect(() => j('nulllll')).toThrow('unexpected');
        expect(() => j('True')).toThrow('unexpected');
        expect(() => j('foo-bar')).toThrow('unexpected');
        expect(() => j('foo bar')).toThrow('unexpected');
        expect(j('false//hello')).toEqual(false);
    });
    test('trivia', () => {
        expect(j(' ')).toEqual(undefined);
        expect(j('  \t  ')).toEqual(undefined);
        expect(j('  \t  \n  \t  ')).toEqual(undefined);
        expect(j('\r\n')).toEqual(undefined);
        expect(j('\r')).toEqual(undefined);
        expect(j('\n')).toEqual(undefined);
        expect(j('\n\r')).toEqual(undefined);
        expect(j('\n   \n')).toEqual(undefined);
    });
    test('literals', () => {
        expect(j('true')).toEqual(true);
        expect(j('false')).toEqual(false);
        expect(j('null')).toEqual(null);
        expect(j('"foo"')).toEqual('foo');
        expect(j('"\\"-\\\\-\\/-\\b-\\f-\\n-\\r-\\t"')).toEqual('"-\\-/-\b-\f-\n-\r-\t');
        expect(j('"\\u00DC"')).toEqual('Ü');
        expect(j('9')).toEqual(9);
        expect(j('-9')).toEqual(-9);
        expect(j('0.129')).toEqual(0.129);
        expect(j('23e3')).toEqual(23e3);
        expect(j('1.2E+3')).toEqual(1.2E+3);
        expect(j('1.2E-3')).toEqual(1.2E-3);
        expect(j('1.2E-3 // comment')).toEqual(1.2E-3);
    });
    test('objects', () => {
        expect(j('{}')).toEqual({});
        expect(j('{ "foo": true }')).toEqual({ foo: true });
        expect(j('{ "bar": 8, "xoo": "foo" }')).toEqual({ bar: 8, xoo: 'foo' });
        expect(j('{ "hello": [], "world": {} }')).toEqual({ hello: [], world: {} });
        expect(j('{ "a": false, "b": true, "c": [ 7.4 ] }')).toEqual({ a: false, b: true, c: [7.4] });
        expect(j('{ "lineComment": "//", "blockComment": ["/*", "*/"], "brackets": [ ["{", "}"], ["[", "]"], ["(", ")"] ] }'))
            .toEqual({ lineComment: '//', blockComment: ['/*', '*/'], brackets: [['{', '}'], ['[', ']'], ['(', ')']] });
        expect(j('{ "hello": [], "world": {} }')).toEqual({ hello: [], world: {} });
        expect(j('{ "hello": { "again": { "inside": 5 }, "world": 1 }}')).toEqual({ hello: { again: { inside: 5 }, world: 1 } });
        expect(j('{ "foo": /*hello*/true }')).toEqual({ foo: true });
        expect(j('{ "": true }')).toEqual({ '': true });
    });
    test('arrays', () => {
        expect(j('[]')).toEqual([]);
        expect(j('[ [],  [ [] ]]')).toEqual([[], [[]]]);
        expect(j('[ 1, 2, 3 ]')).toEqual([1, 2, 3]);
        expect(j('[ { "a": null } ]')).toEqual([{ a: null }]);
    });
    test('objects with errors', () => {
        expect(() => j('{,}')).toThrow();
        expect(() => j('{ "foo": true, }')).toThrow();
        expect(() => j('{ "bar": 8 "xoo": "foo" }')).toThrow();
        expect(() => j('{ ,"bar": 8 }')).toThrow();
        expect(() => j('{ ,"bar": 8, "foo" }')).toThrow();
        expect(() => j('{ "bar": 8, "foo": }')).toThrow();
        expect(() => j('{ 8, "foo": 9 }')).toThrow();
    });
    test('parse: array with errors', () => {
        expect(() => j('[,]')).toThrow();
        expect(() => j('[ 1 2, 3 ]')).toThrow();
        expect(() => j('[ ,1, 2, 3 ]')).toThrow();
        expect(() => j('[ ,1, 2, 3, ]')).toThrow();
    });
    test('errors', () => {
        expect(() => j('1,1')).toThrow();
        expect(() => j('')).toThrow();
    });
    test('disallow comments', () => {
        const nc = jsonic_next_1.Jsonic.make().use(jsonc_1.Jsonc, { disallowComments: true });
        expect(nc('[ 1, 2, null, "foo" ]')).toEqual([1, 2, null, 'foo']);
        expect(nc('{ "hello": [], "world": {} }')).toEqual({ hello: [], world: {} });
        expect(() => nc('{ "foo": /*comment*/ true }')).toThrow();
    });
    test('trailing comma', () => {
        const jc = jsonic_next_1.Jsonic.make().use(jsonc_1.Jsonc, { allowTrailingComma: true });
        expect(jc('{ "hello": [], }')).toEqual({ hello: [] });
        expect(jc('{ "hello": [] }')).toEqual({ hello: [] });
        expect(jc('{ "hello": [], "world": {}, }')).toEqual({ hello: [], world: {} });
        expect(jc('{ "hello": [], "world": {} }')).toEqual({ hello: [], world: {} });
        expect(jc('[ 1, 2, ]')).toEqual([1, 2]);
        expect(jc('[ 1, 2 ]')).toEqual([1, 2]);
        expect(() => j('{ "hello": [], }')).toThrow();
        expect(() => j('{ "hello": [], "world": {}, }')).toThrow();
        expect(() => j('[ 1, 2, ]')).toThrow();
    });
    test('misc', () => {
        expect(j('{ "foo": "bar" }')).toEqual({ "foo": "bar" });
        expect(j('{ "foo": "bar" }')).toEqual({ "foo": "bar" });
        expect(j('{ "foo": "bar" }')).toEqual({ "foo": "bar" });
        expect(j('{ "foo": "bar" }')).toEqual({ "foo": "bar" });
        expect(j('{ "foo": "bar" }')).toEqual({ "foo": "bar" });
        expect(j('{ "foo": "bar" }')).toEqual({ "foo": "bar" });
        expect(j('{ "foo": "bar" }')).toEqual({ "foo": "bar" });
        expect(j('{ "foo": {"bar": 1, "car": 2 } }')).toEqual({ "foo": { "bar": 1, "car": 2 } });
        expect(j('{ "foo": {"bar": 1, "car": 3 } }')).toEqual({ "foo": { "bar": 1, "car": 3 } });
        expect(j('{ "foo": {"bar": 1, "car": 4 } }')).toEqual({ "foo": { "bar": 1, "car": 4 } });
        expect(j('{ "foo": {"bar": 1, "car": 5 } }')).toEqual({ "foo": { "bar": 1, "car": 5 } });
        expect(j('{ "foo": {"bar": 1, "car": 6 } }')).toEqual({ "foo": { "bar": 1, "car": 6 } });
        expect(j('{ "foo": {"bar": 1, "car": 7 } }')).toEqual({ "foo": { "bar": 1, "car": 7 } });
        expect(j('{ "foo": {"bar": 1, "car": 8 }, "goo": {} }')).toEqual({ "foo": { "bar": 1, "car": 8 }, "goo": {} });
        expect(j('{ "foo": {"bar": 1, "car": 9 }, "goo": {} }')).toEqual({ "foo": { "bar": 1, "car": 9 }, "goo": {} });
        expect(() => j('{ "dep": {"bar": 1, "car": ')).toThrow();
        expect(() => j('{ "dep": {"bar": 1,, "car": ')).toThrow();
        expect(() => j('{ "dep": {"bar": "na", "dar": "ma", "car":  } }')).toThrow();
        expect(j('["foo", null ]')).toEqual(["foo", null]);
        expect(j('["foo", null ]')).toEqual(["foo", null]);
        expect(j('["foo", null ]')).toEqual(["foo", null]);
        expect(j('["foo", null ]')).toEqual(["foo", null]);
        expect(j('["foo", null ]')).toEqual(["foo", null]);
        expect(() => j('["foo", null, ]')).toThrow();
        // TODO
        expect(() => j('["foo", null,, ]')).toThrow();
        expect(() => j('[["foo", null,, ],')).toThrow();
        expect(j('true')).toEqual(true);
        expect(j('false')).toEqual(false);
        expect(j('null')).toEqual(null);
        expect(j('23')).toEqual(23);
        expect(j('-1.93e-19')).toEqual(-1.93e-19);
        expect(j('"hello"')).toEqual("hello");
        expect(j('[]')).toEqual([]);
        expect(j('[ 1 ]')).toEqual([1]);
        expect(j('[ 1, "x"]')).toEqual([1, "x"]);
        expect(j('[[]]')).toEqual([[]]);
        expect(j('{ }')).toEqual({});
        expect(j('{ "val": 1 }')).toEqual({ "val": 1 });
        expect(j('{"id": "$", "v": [ null, null] }'))
            .toEqual({ "id": "$", "v": [null, null] });
        expect(() => j('{  "id": { "foo": { } } , }')).toThrow();
        expect(j('{ }')).toEqual({}); //, [{ id: 'onObjectBegin', text: '{', startLine: 0, startCharacter: 0 }, { id: 'onObjectEnd', text: '}', startLine: 0, startCharacter: 2 }]);
        expect(j('{ "foo": "bar" }')).toEqual({ "foo": "bar" });
        expect(j('{ "foo": { "goo": 3 } }')).toEqual({ "foo": { "goo": 3 } });
        expect(j('[]')).toEqual([]);
        expect(j('[ true, null, [] ]')).toEqual([true, null, []]);
        expect(j('[\r\n0,\r\n1,\r\n2\r\n]')).toEqual([0, 1, 2]);
        expect(j('/* g */ { "foo": //f\n"bar" }')).toEqual({ foo: 'bar' });
        expect(j('/* g\r\n */ { "foo": //f\n"bar" }')).toEqual({ foo: 'bar' });
        expect(j('/* g\n */ { "foo": //f\n"bar"\n}')).toEqual({ foo: 'bar' });
        expect(() => j('{"prop1":"foo","prop2":"foo2","prop3":{"prp1":{""}}}')).toThrow();
        expect(j('{ "key1": { "key11": [ "val111", "val112" ] }, "key2": [ { "key21": false, "key22": 221 }, null, [{}] ] }'))
            .toEqual({ "key1": { "key11": ["val111", "val112"] }, "key2": [{ "key21": false, "key22": 221 }, null, [{}]] });
    });
});
//# sourceMappingURL=jsonc.test.js.map