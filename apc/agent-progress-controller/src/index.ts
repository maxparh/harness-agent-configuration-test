import { Plugin } from '@opencode/plugin'
import { appendFile, mkdir } from 'node:fs/promises'
import { join } from 'node:path'
import { ProgressAnalyzer, fingerprint, type Action } from './analyzer.ts'
import { StagnationDetector } from './stagnation.ts'

type RawEvent = Record<string, unknown>
function obj(v: unknown): RawEvent { return v && typeof v === 'object' ? v as RawEvent : {} }
const iso = () => new Date().toISOString()

export default Plugin.define({
  id: 'agent-progress-controller',
  async setup(ctx) {
    const active = ctx.options.mode === 'intervene' || ctx.options.mode === 'active'
    const stagnationEnabled = ctx.options.detectStagnation === true
    const stagnation = new StagnationDetector(Number(ctx.options.minModelCalls ?? 8), Number(ctx.options.minToolCalls ?? 6))
    const threshold = Number(ctx.options.repeatThreshold ?? 3)
    const logDir = String(ctx.options.logDirectory ?? join(process.cwd(), 'apc-logs'))
    const analyzer = new ProgressAnalyzer(Number.isInteger(threshold) && threshold >= 2 ? threshold : 3)
    const pending = new Map<string,number>()
    const interventionCount = new Map<string,number>()
    const stats = new Map<string, {started:string; calls:number; errors:number; alerts:number; interventions:number}>()
    await mkdir(logDir,{recursive:true})
    // Serialize writes to preserve JSONL order within the plugin instance.
    let queue = Promise.resolve()
    function log(row:Record<string,unknown>) {
      queue = queue.then(() => appendFile(join(logDir,'events.jsonl'),JSON.stringify({ts:iso(),...row})+'\n','utf8')).then(()=>undefined)
      queue.catch(e => console.error('[APC] logging failed',e))
      return queue
    }
    const before = await ctx.tool.hook('execute.before', (event) => {
      const e=obj(event)
      const callID=String(e.callID ?? e.id ?? e.toolCallID ?? '')
      if(callID) pending.set(callID,performance.now())
    })
    const after = await ctx.tool.hook('execute.after', async (event) => {
      const e=obj(event)
      // The runtime shape must be verified against installed SDK; don't silently group across sessions.
      const sessionID=String(e.sessionID ?? obj(e.context).sessionID ?? '')
      if(!sessionID) { await log({type:'unattributed_tool_event',fields:Object.keys(e)}); return }
      const id=String(e.callID ?? e.id ?? e.toolCallID ?? '')
      const start=id ? pending.get(id) : undefined
      if(id) pending.delete(id)
      const status = e.status === 'error' ? 'error' : 'completed'
      const result=status === 'error' ? obj(e.error).message ?? e.error : e.result
      const action:Action={sessionID,tool:String(e.tool ?? 'unknown'),input:e.input,status,result,time:iso(),durationMs:start === undefined?undefined:Math.round(performance.now()-start)}
      const s=stats.get(sessionID) ?? {started:iso(),calls:0,errors:0,alerts:0,interventions:0}
      s.calls++;if(status==='error')s.errors++
      stats.set(sessionID,s)
      analyzer.tick(sessionID)
      if (stagnationEnabled) stagnation.onTool(sessionID, action.tool, action.input)
      const alert=analyzer.observe(action)
      await log({type:'tool',sessionID,tool:action.tool,status,inputHash:fingerprint(action.input),outputHash:fingerprint(action.result),durationMs:action.durationMs,callID:id || undefined})
      if(!alert) return
      s.alerts++
      await log({type:'alert',sessionID,...alert})
      if(!active) return
      const count=interventionCount.get(sessionID) ?? 0
      if(count >= 2) return // no endless instructions; MVP does not forcibly interrupt
      const value = alert.kind === 'identical_error'
        ? 'APC detected repeated identical tool errors. Do not retry unchanged conditions. Identify a distinct remedy or report the environment blocker to the user.'
        : 'APC detected repeated identical tool actions/results. Reassess the current plan; choose a materially different next action or explain the blocker.'
      try {
        await ctx.session.instructions.entry.put({sessionID,key:'apc-progress',value})
        interventionCount.set(sessionID,count+1);s.interventions++
        await log({type:'intervention',sessionID,kind:alert.kind,index:count+1})
      } catch(error) { await log({type:'intervention_error',sessionID,error:String(error)}) }
    })
    const contextHook = await ctx.session.hook('context', async (event) => {
      if (!stagnationEnabled) return
      const result = stagnation.onModelCall(event.sessionID)
      if (!result.trigger) return
      await log({type:'stagnation_detected',sessionID:event.sessionID,modelCalls:result.modelCalls,toolCalls:result.tools,observedWrites:result.writes})
      if (!active) return
      const instruction = 'APC: No file-write tool has been observed after multiple implementation steps. This is a development task. On this step, make one concrete implementation change using a write/edit tool, or explicitly state the specific blocker preventing any code change. Avoid repeating a high-level plan. Do not make changes merely to satisfy this instruction if implementation is already complete or no code change is justified.'
      event.system.push({type:'text',text:instruction})
      await log({type:'intervention',sessionID:event.sessionID,kind:'no_observed_write',delivery:'context_hook',modelCalls:result.modelCalls,toolCalls:result.tools})
    })
    const compaction = await ctx.session.hook('compaction', async (event) => {
      await log({type:'compaction_hook',sessionID:event.sessionID})
    })
    await log({type:'plugin_started',mode:active?'intervene':'observe',version:ctx.app.version})
    return async () => {
      await before.dispose();await after.dispose();await contextHook.dispose();await compaction.dispose()
      for(const [sessionID,summary] of stats) await log({type:'summary',sessionID,...summary})
      await queue
    }
  },
})
