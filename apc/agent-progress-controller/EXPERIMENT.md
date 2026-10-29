# APC experimental active intervention (0.2.0 pilot)

- Mode `observe`: event logging and detection only.
- Mode `intervene`: same detector plus ONE system instruction on the next primary agent-loop context request when the selected threshold is reached.
- Configuration option `detectStagnation: true` activates the development-task-specific heuristic. Defaults: `minModelCalls: 8`, `minToolCalls: 6`.
- Proxy definition: at least N model-context requests and M completed tool calls in the current session with no observed write/edit/apply_patch tool. It does **not** identify planning text. Shell-initiated writes are not detected; use normal edit/write tools for the pilot. A completed implementation with no edits is a false-positive risk.
- Event types: `stagnation_detected` and `intervention` with `delivery: context_hook`.
- Before scientific A/B runs, test intervention on a controlled task and independently verify agent behavior. Single-shot injection is not proof the model obeyed. Keep both arms equally instrumented via observe vs intervene; compare equal fresh sessions and clean worktree, record manual independent task success/test metrics.
- `intervene` with `detectStagnation: true` records repeated-call alerts but does not intervene on them, isolating the single no-observed-write mechanism.
