import test from 'node:test'
import assert from 'node:assert/strict'
import { StagnationDetector } from '../src/stagnation.ts'
test('normal exploration does not trigger too early',()=>{
 const d=new StagnationDetector(4,3)
 for(let i=0;i<3;i++) {d.onTool('s','read',{file:i}); assert.equal(d.onModelCall('s').trigger,false)}
})
test('stagnation proxy triggers once',()=>{
 const d=new StagnationDetector(4,3)
 for(let i=0;i<3;i++) {d.onTool('s','read',{file:i});d.onModelCall('s')}
 assert.equal(d.onModelCall('s').trigger,true)
 assert.equal(d.onModelCall('s').trigger,false)
})
test('observed edit suppresses intervention',()=>{
 const d=new StagnationDetector(3,2)
 d.onTool('s','read',{});d.onTool('s','edit',{})
 for(let i=0;i<5;i++)assert.equal(d.onModelCall('s').trigger,false)
})
test('per-session isolation',()=>{
 const d=new StagnationDetector(2,1)
 d.onTool('s','read',{});d.onModelCall('s')
 assert.equal(d.onModelCall('s').trigger,true)
 assert.equal(d.onModelCall('other').trigger,false)
})
