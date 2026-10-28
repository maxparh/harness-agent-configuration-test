# Reconstructed experimental launch configuration (not the original start.ps1).
# Set the path to your downloaded Q3_K_M GGUF before launching.
$ErrorActionPreference = 'Stop'
$llamaServer = 'C:\llama-cuda\llama-server.exe'
$modelPath = $env:QWEN_GGUF_PATH

if (-not $modelPath) {
    throw 'Set QWEN_GGUF_PATH to the absolute path of your Q3_K_M .gguf file.'
}
if (-not (Test-Path -LiteralPath $llamaServer)) {
    throw "llama-server not found: $llamaServer"
}
if (-not (Test-Path -LiteralPath $modelPath)) {
    throw "GGUF model not found: $modelPath"
}

& $llamaServer `
    --model $modelPath `
    --alias 'qwen38-local' `
    --host '127.0.0.1' `
    --port 8080 `
    --ctx-size 32768 `
    --n-gpu-layers auto `
    --flash-attn on `
    --cache-type-k q8_0 `
    --cache-type-v q8_0 `
    --parallel 1 `
    --jinja
