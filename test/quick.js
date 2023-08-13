const ini = require('ini')

const { Jsonic } = require('@jsonic/jsonic-next')
const { Debug } = require('@jsonic/jsonic-next/debug')

// console.log(Debug)

const { Ini } = require('..')

const j = Jsonic.make().use(Debug, { trace: true }).use(Ini, {})

console.dir(
  j(`
a=1

`),
)

// console.dir(j(`
// ; comment
// e[]=11
// e[]=22
// a=1
// b=2
// c c=3
// [q]
// d=4
// `))

// console.dir(j(`
// a.a=A
// s="\\n"
// q="'Q'"
// qq='Q'
// "[]"='[]'
// `))

// console.dir(j(`
// a0=0
// a[]=1
// a[]=2
// a=3
// a1=33
// b[]=11
// c=44
// [q]
// w[]=55
// w[]=66
// `))

// console.dir(j(`
// [A]
// [B]
// [C]
// `))

// console.dir(j(`
// t=true
// [a.b]
// c=1
// d=[1,2]
// e=]1,2[
// `))

/*
console.dir(j(`
; comment
a = 1
b = x
c = y y


[d]
e = 2

[f]
g = '1'

[f1]
[f2]


[h.i]
#j = [3,4]
k = true

[l.m.n.o]
p = "P"
q = {x:5}
u = v = 5
w = "{y:{z:6}}"
aa = 7

`), { depth: null })



console.log(ini.decode(`
a = "{x:1}"
b = '{"x":1}'
aa = "1"
bb = '1'
`))
*/

console.log(ini.decode(`=1`))
