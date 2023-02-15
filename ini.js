"use strict";
/* Copyright (c) 2021-2023 Richard Rodger, MIT License */
Object.defineProperty(exports, "__esModule", { value: true });
exports.Ini = void 0;
const hoover_1 = require("@jsonic/hoover");
function Ini(jsonic, options) {
    jsonic.use(hoover_1.Hoover, {
        lex: {
            order: 7.5e6 // before text, after string, number
        },
        block: {
            endofline: {
                start: {
                    rule: {
                        parent: {
                            include: ['pair', 'elem']
                        },
                    }
                },
                end: {
                    fixed: ['\n', '\r\n', '#', ';', '']
                },
                escapeChar: '\\',
                escape: {
                    '#': '#',
                    ';': ';',
                    '\\': '\\',
                },
                trim: true,
            }
        }
    });
    jsonic.options({
        rule: {
            start: 'ini',
            exclude: 'jsonic',
        },
        lex: {
            emptyResult: {},
        },
        fixed: {
            token: {
                '#EQ': '=',
                '#DOT': '.',
            },
        },
        // FIX: breaks, as HV takes over
        // number: {
        //   lex: false
        // },
        comment: {
            def: {
                slash: null,
                multi: null,
                semi: { line: true, start: ';', lex: true },
            },
        },
    });
    const { ZZ, ST, NR, OS, CS, CL, EQ, DOT, CA, OB } = jsonic.token;
    const KEY = jsonic.tokenSet.VAL;
    jsonic.rule('ini', (rs) => {
        rs.bo((r) => {
            r.node = {};
        }).open([
            { s: [KEY, EQ], p: 'table', b: 2 },
            { s: [OS, KEY], p: 'table', b: 2 },
            { s: [ZZ] },
        ]);
    });
    jsonic.rule('table', (rs) => {
        rs
            .bo((r) => {
            r.node = r.parent.node;
            if (r.prev.use.dive) {
                let dive = r.prev.use.dive;
                for (let dI = 0; dI < dive.length; dI++) {
                    console.log('DIVE', dI, dive[dI], dive);
                    r.node = r.node[dive[dI]] = {};
                }
            }
        })
            .open([
            { s: [KEY, EQ], p: 'map', b: 2 },
            { s: [OS, KEY], p: 'dive', b: 1 },
            { s: [CS], p: 'map' },
        ])
            .bc((r) => {
            Object.assign(r.node, r.child.node);
        })
            .close([
            { s: [OS], r: 'table', b: 1 },
            { s: [CS], r: 'table', a: (r) => r.use.dive = r.child.use.dive },
            { s: [ZZ] },
        ]);
    });
    // TODO: maybe backport this to toml?
    jsonic.rule('dive', (rs) => {
        rs
            .open([
            {
                s: [KEY, DOT],
                a: (r) => (r.use.dive = r.parent.use.dive || []).push(r.o0.val),
                p: 'dive'
            },
            {
                s: [KEY],
                a: (r) => (r.use.dive = r.parent.use.dive || []).push(r.o0.val)
            }
        ])
            .close([{ s: [CS], b: 1 }]);
    });
    jsonic.rule('map', (rs) => {
        rs
            .open([
            // Pair from implicit map.
            {
                s: [KEY, EQ],
                c: (r) => 'table' === r.parent.name,
                p: 'pair',
                b: 2
            },
            {
                s: [KEY],
                c: (r) => 'table' === r.parent.name,
                p: 'pair',
                b: 1
            },
        ], { append: true })
            .close([
            { s: [OS], b: 1 },
            { s: [ZZ] }
        ]);
    });
    jsonic.rule('pair', (rs) => {
        rs
            .open([
            {
                s: [KEY, EQ],
                c: (r) => 'table' === r.parent.parent.name,
                p: 'val',
                u: { pair: true },
                a: (r) => r.use.key = r.o0.val
            },
            {
                s: [KEY],
                c: (r) => 'table' === r.parent.parent.name,
                a: (r) => r.parent.node[r.o0.val] = true
            },
        ])
            .close([
            {
                s: [KEY, CL],
                c: (r) => 'table' === r.parent.parent.name,
                e: (r) => {
                    return r.c1;
                }
            },
            { s: [KEY], b: 1, r: 'pair' },
            { s: [OS], b: 1 },
        ]);
    });
}
exports.Ini = Ini;
//# sourceMappingURL=ini.js.map