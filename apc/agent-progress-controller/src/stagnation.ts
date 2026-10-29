/** Deterministic proxy for lack of implementation progress, not a classifier of model reasoning. */
export class StagnationDetector {
  private sessions = new Map<string, { modelCalls: number; tools: number; writes: number; fired: boolean; started: number }>()
  private readonly minModelCalls: number
  private readonly minToolCalls: number
  constructor(minModelCalls = 8, minToolCalls = 6) { this.minModelCalls = minModelCalls; this.minToolCalls = minToolCalls }
  onTool(sessionID: string, tool: string, input: unknown) {
    const s = this.get(sessionID)
    s.tools++
    if (/^(write|edit|apply_patch|patch|multiedit)$/i.test(tool)) s.writes++
    // Shell writes cannot be reliably inferred from their command text.
  }
  onModelCall(sessionID: string): {trigger: boolean; modelCalls: number; tools: number; writes: number} {
    const s = this.get(sessionID)
    s.modelCalls++
    const trigger = !s.fired && s.writes === 0 && s.modelCalls >= this.minModelCalls && s.tools >= this.minToolCalls
    if (trigger) s.fired = true
    return {trigger,modelCalls:s.modelCalls,tools:s.tools,writes:s.writes}
  }
  private get(id: string) {
    let s = this.sessions.get(id)
    if (!s) { s = {modelCalls:0,tools:0,writes:0,fired:false,started:Date.now()}; this.sessions.set(id,s) }
    return s
  }
}
