/* Copyright (c) 2021-2025 Richard Rodger, MIT License */

// Import Jsonic types used by plugin.
import { Jsonic, RuleSpec, NormAltSpec, Lex, makePoint, Token } from 'jsonic'
import { Hoover } from '@jsonic/hoover'

type IniOptions = {
  multiline?: {
    // Character before newline indicating continuation. Default: '\\'.
    // Set to false to disable backslash continuation.
    continuation?: string | false
    // When true, a continuation line must be indented (leading whitespace).
    // Indented lines continue the previous value even without a continuation char.
    indent?: boolean
  } | boolean
}

function Ini(jsonic: Jsonic, _options: IniOptions) {
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

  // Multiline value support via custom lex matcher.
  // Newlines terminate values at the lex level (Hoover's endofline block),
  // so continuation must be handled by a higher-priority lex matcher that
  // replaces endofline in value contexts.
  const multiline = true === _options.multiline ? {} : _options.multiline
  if (multiline) {
    const continuation: string | false =
      multiline.continuation !== undefined ? multiline.continuation : '\\'
    const indent = multiline.indent || false
    const HV_TIN = jsonic.token('#HV') as number

    jsonic.options({
      lex: {
        match: {
          multiline: {
            // Lower order than Hoover (8.5e6) so this runs first.
            order: 8.4e6,
            make: () => {
              return function multilineMatcher(lex: Lex): Token | undefined {
                // Only match in value context during rule open state
                // (same as Hoover endofline block, which defaults to state 'o').
                let ctx = (lex as any).ctx
                let parentName = ctx?.rule?.parent?.name
                if (parentName !== 'pair' && parentName !== 'elem') {
                  return undefined
                }
                if (ctx?.rule?.state !== 'o') {
                  return undefined
                }

                let src = lex.src
                let sI = lex.pnt.sI
                let rI = lex.pnt.rI
                let cI = lex.pnt.cI
                let startI = sI
                let chars: string[] = []

                while (sI < src.length) {
                  let c = src[sI]

                  // Check for comment characters (end value).
                  if (c === '#' || c === ';') break

                  // Check for backslash continuation before newline.
                  if (false !== continuation && c === continuation) {
                    if (src[sI + 1] === '\n') {
                      // \<LF> continuation
                      sI += 2; rI++; cI = 0
                      // Consume leading whitespace on continuation line.
                      while (sI < src.length &&
                        (src[sI] === ' ' || src[sI] === '\t')) {
                        sI++; cI++
                      }
                      continue
                    }
                    if (src[sI + 1] === '\r' && src[sI + 2] === '\n') {
                      // \<CR><LF> continuation
                      sI += 3; rI++; cI = 0
                      while (sI < src.length &&
                        (src[sI] === ' ' || src[sI] === '\t')) {
                        sI++; cI++
                      }
                      continue
                    }
                  }

                  // Check for newline.
                  if (c === '\n' || (c === '\r' && src[sI + 1] === '\n')) {
                    // Indent continuation: next line starts with whitespace.
                    if (indent) {
                      let nextI = c === '\r' ? sI + 2 : sI + 1
                      if (nextI < src.length &&
                        (src[nextI] === ' ' || src[nextI] === '\t')) {
                        rI++; cI = 0
                        sI = nextI
                        // Consume leading whitespace.
                        while (sI < src.length &&
                          (src[sI] === ' ' || src[sI] === '\t')) {
                          sI++; cI++
                        }
                        chars.push(' ')
                        continue
                      }
                    }

                    // Normal newline: end value and consume the newline.
                    if (c === '\r') { sI += 2 } else { sI++ }
                    rI++; cI = 0
                    break
                  }

                  // Handle escape sequences (same as Hoover endofline block).
                  if (c === '\\' && sI + 1 < src.length) {
                    let next = src[sI + 1]
                    if (next === '#' || next === ';') {
                      chars.push(next)
                      sI += 2; cI += 2
                      continue
                    }
                    if (next === '\\') {
                      chars.push('\\')
                      sI += 2; cI += 2
                      continue
                    }
                  }

                  chars.push(c)
                  sI++; cI++
                }

                let val: string | undefined = chars.join('').trim()

                let pnt = makePoint(lex.pnt.len, sI, rI, cI)
                let tkn = lex.token(
                  HV_TIN, val, src.substring(startI, sI), pnt)
                tkn.use = { block: 'endofline' }

                lex.pnt.sI = sI
                lex.pnt.rI = rI
                lex.pnt.cI = cI

                return tkn
              }
            }
          }
        }
      }
    })
  }

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

      if (r.prev.u.dive) {
        let dive = r.prev.u.dive
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
        { s: [CS], r: 'table', a: (r) => (r.u.dive = r.child.u.dive) },
        { s: [ZZ] },
      ])
  })

  jsonic.rule('dive', (rs: RuleSpec) => {
    rs.open([
      {
        s: [DK, DOT],
        a: (r) => (r.u.dive = r.parent.u.dive || []).push(r.o0.val),
        p: 'dive',
      },
      {
        s: [DK],
        a: (r) => (r.u.dive = r.parent.u.dive || []).push(r.o0.val),
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
      { append: true },
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
            r.u.ini_array = r.node[key]
          } else {
            r.u.key = key
            if (2 < key.length && key.endsWith('[]')) {
              key = r.u.key = key.slice(0, -2)
              r.node[key] = r.u.ini_array = Array.isArray(r.node[key])
                ? r.node[key]
                : undefined === r.node[key]
                  ? []
                  : [r.node[key]]
            } else {
              r.u.pair = true
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
    ]).close([
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
      },
    ).ac((r) => {
      if (ST === r.o0.tin && "'" === r.o0.src[0]) {
        try {
          r.node = JSON.parse(r.node)
        } catch (e) {
          // Invalid JSON, just accept val as given
        }
      }

      if (null != r.prev.u.ini_prev) {
        r.prev.node = r.node = r.prev.o0.src + r.node
      } else if (r.parent.u.ini_array) {
        r.parent.u.ini_array.push(r.node)
      }
    })
  })
}

export { Ini }

export type { IniOptions }
