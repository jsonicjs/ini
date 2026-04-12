#!/usr/bin/env node

// Embeds ini-grammar.jsonic into go/ini.go as a Go raw string constant.
// Run via: npm run embed-go

const fs = require('fs')
const path = require('path')

const grammarFile = path.join(__dirname, 'ini-grammar.jsonic')
const goFile = path.join(__dirname, 'go', 'ini.go')

const grammar = fs.readFileSync(grammarFile, 'utf8')

// Go raw strings (backtick-delimited) cannot contain backticks.
if (grammar.includes('`')) {
  console.error('Error: grammar file contains backticks, cannot embed in Go raw string')
  process.exit(1)
}

let goSrc = fs.readFileSync(goFile, 'utf8')

const BEGIN = '// --- BEGIN EMBEDDED ini-grammar.jsonic ---'
const END = '// --- END EMBEDDED ini-grammar.jsonic ---'

const beginIdx = goSrc.indexOf(BEGIN)
const endIdx = goSrc.indexOf(END)

if (beginIdx === -1 || endIdx === -1) {
  console.error('Error: embedding markers not found in go/ini.go')
  process.exit(1)
}

const replacement = BEGIN + '\n' +
  'const grammarText = `\n' + grammar + '`\n' +
  END

goSrc = goSrc.substring(0, beginIdx) + replacement + goSrc.substring(endIdx + END.length)

fs.writeFileSync(goFile, goSrc)
