# Agent Progress Controller — MVP 0.1.0

Local OpenCode V2 server plugin. Requires OpenCode 2.2.0 and Node >= 22 for included tests. No cloud APIs, no changes to source code. `mode: observe` is default.

## Windows setup
1. Unzip into `C:\APC\agent-progress-controller`.
2. In PowerShell: `cd C:\APC\agent-progress-controller; npm install; npm run test; npm run check`.
3. Backup your **existing** `opencode.jsonc` (or `opencode.json`). Merge the `plugins` item from `examples/opencode.jsonc` into the existing configuration; never replace your existing model config. Adjust path and output folder.
4. Restart OpenCode Desktop; open a **new session**. Run a small task with several tool calls.
5. Inspect `C:\APC\logs\events.jsonl` (`Get-Content C:\APC\logs\events.jsonl -Tail 20`).
6. If the plugin loads and tool events carry correct `sessionID`, change `mode` to `active` only for controlled test runs and restart OpenCode.
7. Disable by removing the plugin entry from `plugins` and restarting OpenCode. Logs stay outside the code project.

## Behavior and limitations
- Hooks: tool.execute.before/after; session.compaction, session.instructions.entry.put, sourced from OpenCode V2 API docs.
- Detects **consecutive identical calls and outputs**, not semantic stagnation. Diverse file reads are not flagged. No intervention if no session ID is present. Logs a `unattributed_tool_event` entry instead; investigate SDK shape before enabling active mode.
- Stores hashes of inputs/outputs rather than raw contents, so errors are not included verbatim in JSONL. Fingerprint collisions are computationally negligible. Logs may still reveal tool names/session IDs: protect logs.
- Instrument duration may be missing if before/after call IDs are unavailable or differ. The compaction hook reports invoked compaction flows, not necessarily successful compaction. Token usage and final product status are **not** implemented in this MVP.
- First two warnings per session can become session instructions in active mode; no hard-stop or tool blocking is implemented. The controller deliberately does not change project files.
- Test runs here exercise pure analyzer only; integration compatibility must be verified on the actual installed OpenCode package. In particular, examine the installed types of ToolExecuteBefore/Completed/Failed if events are unattributed.

## Suggested experiment
Before A/B comparison, use a short pilot to verify events, log order, intervention delivery, and no changes to model parameters. Keep the same prompts, restored Git revision, session count, time budget, inference server configuration, and active/passive instrumentation parity. Record outcome and tests independently.

Documentation: https://opencode.ai/v2/docs/build/plugins/server/ and https://opencode.ai/v2/docs/build/plugins/
