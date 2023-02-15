const { Jsonic } = require('@jsonic/jsonic-next')
const { Debug } = require('@jsonic/jsonic-next/debug')

console.log(Debug)

const { Jsonc } = require('..')

const jsonc = Jsonic.make()
  .use(Debug, {
    trace: true,
  })
  .use(Jsonc, {})

// console.log(jsonc.internal().config)

// console.dir(jsonc(`// comment`),{depth:null})
// console.dir(jsonc('"\\v"'),{depth:null})
console.dir(jsonc('.0'), { depth: null })
