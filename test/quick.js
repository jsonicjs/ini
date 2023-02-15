const { Jsonic } = require('@jsonic/jsonic-next')
const { Debug } = require('@jsonic/jsonic-next/debug')

console.log(Debug)

const { Ini } = require('..')

const ini = Jsonic.make()
  .use(Debug, {
    trace: true,
  })
  .use(Ini, {})

console.dir(ini(`
; comment
a = 1
b = x
c = y y


[d]
e = 2

[f]
# x:11
g = 'G'
# x:12


[h.i]
j = [3,4]
k = true

[l.m.n.o]
p = "P"
q = {x:5}
u = v = 5
w = {y:{z:6}}
aa = 7

`), { depth: null })



