import test from 'node:test'
import assert from 'node:assert/strict'
import { ProgressAnalyzer } from '../src/analyzer.ts'
const event = (tool, input, result, status='completed') => ({sessionID:'s1',tool,input,result,status,time:new Date().toISOString()})
function step(a,e) { a.tick(e.sessionID); return a.observe(e) }
test('normal diverse reads do not alert',()=>{
 const a=new ProgressAnalyzer()
 for(const file of ['a.go','b.go','c.go','d.go']) assert.equal(step(a,event('read',{file},'contents')),undefined)
})
test('identical error triggers on third attempt',()=>{
 const a=new ProgressAnalyzer()
 const e=event('bash',{command:'go test -race ./...'},'CGO failure','error')
 assert.equal(step(a,e),undefined);assert.equal(step(a,e),undefined)
 assert.equal(step(a,e)?.kind,'identical_error')
})
test('changed command resets consecutive repetition',()=>{
 const a=new ProgressAnalyzer()
 const e=event('bash',{command:'go test'},'fail','error')
 step(a,e);step(a,event('bash',{command:'gcc --version'},'ok'))
 assert.equal(step(a,e),undefined)
})
test('read-cycle detected with exact repeated content',()=>{
 const a=new ProgressAnalyzer();const e=event('read',{file:'x.go'},'x')
 step(a,e);step(a,e);assert.equal(step(a,e)?.kind,'read_cycle')
})
