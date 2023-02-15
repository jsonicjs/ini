"use strict";
/* Copyright (c) 2021-2023 Richard Rodger, MIT License */
Object.defineProperty(exports, "__esModule", { value: true });
exports.Jsonc = void 0;
function Jsonc(jsonic, options) {
    jsonic.options({
        text: {
            lex: false,
        },
        number: {
            hex: false,
            oct: false,
            bin: false,
            sep: null,
            exclude: /^\./,
        },
        string: {
            chars: '"',
            multiChars: '',
            allowUnknown: false,
            escape: {
                v: null,
            },
        },
        comment: {
            lex: true !== options.disallowComments,
        },
        map: {
            extend: false,
        },
        lex: {
            empty: false,
        },
        rule: {
            finish: false,
            include: 'jsonc,json' + (options.allowTrailingComma ? ',comma' : ''),
        },
    });
    const { ZZ } = jsonic.token;
    jsonic.rule('val', (rs) => {
        rs.open([
            {
                s: [ZZ],
                g: 'jsonc',
            },
        ]);
    });
}
exports.Jsonc = Jsonc;
//# sourceMappingURL=jsonc.js.map