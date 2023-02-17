/* Copyright (c) 2021-2023 Richard Rodger, MIT License */

// Import Jsonic types used by plugin.
import { Jsonic, Rule, RuleSpec, NormAltSpec, Lex } from '@jsonic/jsonic-next'
import { Hoover } from '@jsonic/hoover'

type IniOptions = {
  allowTrailingComma?: boolean
  disallowComments?: boolean
}

function Ini(jsonic: Jsonic, options: IniOptions) {
  jsonic.use(Hoover, {
    lex: {
      order: 8.5e6,
    },
    block: {
      endofline: {
        start: {
          rule: {
            parent: {
              include: ['pair', 'elem'],
            },
          },
        },
        end: {
          fixed: ['\n', '\r\n', '#', ';', ''],
          consume: ['\n', '\r\n'],
        },
        escapeChar: '\\',
        escape: {
          '#': '#',
          ';': ';',
          '\\': '\\',
        },
        preserveEscapeChar: true,
        trim: true,
      },
      key: {
        token: '#HK',
        start: {
          rule: {
            current: {
              exclude: ['dive'],
            },
            state: 'oc',
          },
        },
        end: {
          fixed: ['=', '\n', '\r\n', '#', ';', ''],
          consume: false,
        },
        escape: {
          '#': '#',
          ';': ';',
          '\\': '\\',
        },
        trim: true,
      },
      divekey: {
        token: '#DK',
        start: {
          rule: {
            current: {
              include: ['dive'],
            },
          },
        },
        end: {
          fixed: [']', '.'],
          consume: false,
        },
        escapeChar: '\\',
        escape: {
          ']': ']',
          '.': '.',
          '\\': '\\',
        },
        allowUnknownEscape: true,
        trim: true,
      },
    },
  })

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
        '#OB': null,
        '#CB': null,
        '#CL': null,
      },
    },

    line: {
      check: (lex: Lex) => {
        if ('val' === lex.ctx.rule.name) {
          return { done: true, token: undefined }
        }
      },
    },
    number: {
      lex: false,
    },
    string: {
      lex: true,
      chars: `'"`,
      abandon: true,
    },
    text: {
      lex: false,
    },

    comment: {
      def: {
        hash: { eatline: true },
        slash: null,
        multi: null,
        semi: { line: true, start: ';', lex: true, eatline: true },
      },
    },
  })

  const { ZZ, ST, VL, OS, CS, CL, EQ, DOT, HV, HK, DK } = jsonic.token

  const KEY = [HK, ST, VL]

  jsonic.rule('ini', (rs: RuleSpec) => {
    rs.bo((r) => {
      r.node = {}
    }).open([
      { s: [OS], p: 'table', b: 1 },
      { s: [KEY, EQ], p: 'table', b: 2 },
      { s: [HV, OS], p: 'table', b: 2 },
      { s: [ZZ] },
    ])
  })

  jsonic.rule('table', (rs: RuleSpec) => {
    rs.bo((r) => {
      r.node = r.parent.node

      if (r.prev.use.dive) {
        let dive = r.prev.use.dive
        for (let dI = 0; dI < dive.length; dI++) {
          r.node = r.node[dive[dI]] = r.node[dive[dI]] || {}
        }
      }
    })
      .open([
        { s: [OS], p: 'dive' },
        { s: [KEY, EQ], p: 'map', b: 2 },
        { s: [HV, OS], p: 'map', b: 2 },
        { s: [CS], p: 'map' },
        { s: [ZZ] },
      ])
      .bc((r) => {
        Object.assign(r.node, r.child.node)
      })
      .close([
        { s: [OS], r: 'table', b: 1 },
        { s: [CS], r: 'table', a: (r) => (r.use.dive = r.child.use.dive) },
        { s: [ZZ] },
      ])
  })

  // TODO: maybe backport this to toml?
  jsonic.rule('dive', (rs: RuleSpec) => {
    rs.open([
      {
        s: [DK, DOT],
        a: (r) => (r.use.dive = r.parent.use.dive || []).push(r.o0.val),
        p: 'dive',
      },
      {
        s: [DK],
        a: (r) => (r.use.dive = r.parent.use.dive || []).push(r.o0.val),
      },
    ]).close([{ s: [CS], b: 1 }])
  })

  jsonic.rule('map', (rs: RuleSpec) => {
    rs.open(
      [
        // Pair from implicit map.
        {
          s: [KEY, EQ],
          c: (r) => 'table' === r.parent.name,
          p: 'pair',
          b: 2,
        },

        {
          s: [KEY],
          c: (r) => 'table' === r.parent.name,
          p: 'pair',
          b: 1,
        },
      ],
      { append: true }
    ).close([{ s: [OS], b: 1 }, { s: [ZZ] }])
  })

  jsonic.rule('pair', (rs: RuleSpec) => {
    rs.open([
      {
        s: [KEY, EQ],
        c: (r) => 'table' === r.parent.parent.name,
        p: 'val',
        a: (r) => {
          let key = '' + r.o0.val
          if (Array.isArray(r.node[key])) {
            r.use.ini_array = r.node[key]
          } else {
            r.use.key = key
            if (2 < key.length && key.endsWith('[]')) {
              key = r.use.key = key.slice(0, -2)
              r.node[key] = r.use.ini_array = Array.isArray(r.node[key])
                ? r.node[key]
                : undefined === r.node[key]
                ? []
                : [r.node[key]]
            } else {
              r.use.pair = true
            }
          }
        },
      },

      // Special case: key by itself means key=true
      {
        s: [HK],
        c: (r) => 'table' === r.parent.parent.name,
        a: (r) => {
          let key = r.o0.val
          if ('string' === typeof key && 0 < key.length) {
            r.parent.node[key] = true
          }
        },
      },
    ])
    .close([
      {
        s: [KEY, CL],
        c: (r) => 'table' === r.parent.parent.name,
        e: (r) => {
          return r.c1
        },
      },
      { s: [KEY], b: 1, r: 'pair' },
      { s: [OS], b: 1 },
    ])
  })

  jsonic.rule('val', (rs: RuleSpec) => {
    rs.open(
      [
        // Since OS,CS are fixed tokens, concat them with string value
        // if they appear as first char in a RHS value.
        {
          s: [[OS, CS]],
          r: 'val',
          u: { ini_prev: true },
        },

        { s: [ZZ], a: (r) => (r.node = '') },
      ],
      {
        custom: (alts: NormAltSpec[]) =>
          alts.filter((alt: NormAltSpec) => alt.g.join() !== 'json,list'),
      }
    ).ac((r) => {
      if (ST === r.o0.tin && "'" === r.o0.src[0]) {
        try {
          r.node = JSON.parse(r.node)
        } catch (e) {
          // Invalid JSON, just accept val as given
        }
      }

      if (null != r.prev.use.ini_prev) {
        r.prev.node = r.node = r.prev.o0.src + r.node
      } else if (r.parent.use.ini_array) {
        r.parent.use.ini_array.push(r.node)
      }
    })
  })
}

export { Ini }

export type { IniOptions }
