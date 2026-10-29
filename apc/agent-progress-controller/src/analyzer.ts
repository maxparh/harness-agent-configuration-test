import { createHash } from 'node:crypto'

export type Action = { sessionID: string; tool: string; input: unknown; status: 'completed'|'error'; result: unknown; time: string; durationMs?: number }
export type Alert = { kind: 'identical_error'|'identical_call'|'read_cycle'; level: 1|2|3|4; description: string; count: number }
type Seen = { key: string; output: string; tool: string; status: string }
export const fingerprint = (value: unknown): string => createHash('sha256').update(stable(value)).digest('hex')
function stable(value: unknown): string {
  if (value === undefined) return 'undefined'
  if (value === null || typeof value !== 'object') return JSON.stringify(value) ?? String(value)
  if (Array.isArray(value)) return '[' + value.map(stable).join(',') + ']'
  return '{' + Object.keys(value as object).sort().map(k => JSON.stringify(k)+':'+stable((value as Record<string, unknown>)[k])).join(',') + '}'
}
export class ProgressAnalyzer {
  private history = new Map<string, Seen[]>()
  private lastAlerts = new Map<string, number>()
  private threshold: number
  private window: number
  private cooldown: number
  constructor(threshold = 3, window = 12, cooldown = 6) { this.threshold=threshold; this.window=window; this.cooldown=cooldown }
  observe(a: Action): Alert | undefined {
    const items = this.history.get(a.sessionID) ?? []
    const current: Seen = { key: fingerprint([a.tool,a.input]), output: fingerprint([a.status,a.result]), tool: a.tool.toLowerCase(), status: a.status }
    items.push(current)
    if (items.length > this.window) items.shift()
    this.history.set(a.sessionID, items)
    const last = items.slice(-this.threshold)
    if (last.length !== this.threshold) return
    const exact = last.every(x => x.key === current.key && x.output === current.output)
    const isRead = /(^|[._-])(read|grep|glob|search)([._-]|$)/.test(current.tool)
    let kind: Alert['kind'] | undefined
    if (exact && current.status === 'error') kind='identical_error'
    else if (exact && !isRead) kind='identical_call'
    else if (exact && isRead) kind='read_cycle'
    if (!kind) return
    const identity = a.sessionID+':'+kind+':'+current.key+':'+current.output
    const lastAt = this.lastAlerts.get(identity) ?? -1e9
    const n = (this.counts.get(a.sessionID) ?? 0)
    if (n-lastAt < this.cooldown) return
    this.lastAlerts.set(identity,n)
    return {kind,level:1,count:this.threshold,description:kind === 'identical_error'?'Identical tool errors repeated':kind === 'read_cycle'?'Same read repeated without changed result':'Same tool call and result repeated'}
  }
  private counts = new Map<string,number>()
  // Increment once per completed event before analyzing it.
  tick(sessionID:string) { this.counts.set(sessionID,(this.counts.get(sessionID) ?? 0)+1) }
}
