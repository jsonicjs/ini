/* Copyright (c) 2021-2023 Richard Rodger, MIT License */

// Import Jsonic types used by plugin.
import { Jsonic, RuleSpec } from '@jsonic/jsonic-next'

type JsoncOptions = {
  allowTrailingComma?: boolean
  disallowComments?: boolean
}

function Jsonc(jsonic: Jsonic, options: JsoncOptions) {
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
  })

  const { ZZ } = jsonic.token

  jsonic.rule('val', (rs: RuleSpec) => {
    rs.open([
      {
        s: [ZZ],
        g: 'jsonc',
      },
    ])
  })
}

export { Jsonc }

export type { JsoncOptions }
