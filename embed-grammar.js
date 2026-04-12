#!/usr/bin/env node

// Embeds ini-grammar.jsonic into src/ini.ts as a template literal.
// Run via: npm run embed

const fs = require('fs')
const path = require('path')

const grammarFile = path.join(__dirname, 'ini-grammar.jsonic')
const iniTsFile = path.join(__dirname, 'src', 'ini.ts')

const grammar = fs.readFileSync(grammarFile, 'utf8')

// Escape for safe embedding in a JavaScript/TypeScript template literal.
const escaped = grammar
  .replace(/\\/g, '\\\\')
  .replace(/`/g, '\\`')
  .replace(/\$\{/g, '\\${')

let iniTs = fs.readFileSync(iniTsFile, 'utf8')

const BEGIN = '// --- BEGIN EMBEDDED ini-grammar.jsonic ---'
const END = '// --- END EMBEDDED ini-grammar.jsonic ---'

const beginIdx = iniTs.indexOf(BEGIN)
const endIdx = iniTs.indexOf(END)

if (beginIdx === -1 || endIdx === -1) {
  console.error('Error: embedding markers not found in src/ini.ts')
  process.exit(1)
}

const replacement = BEGIN + '\n' +
  'const grammarText = `\n' + escaped + '`\n' +
  END

iniTs = iniTs.substring(0, beginIdx) + replacement + iniTs.substring(endIdx + END.length)

fs.writeFileSync(iniTsFile, iniTs)
