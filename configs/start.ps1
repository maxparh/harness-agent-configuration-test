param(
    [ValidateSet('wiki7b', 'general27b')]
    [string]$Profile = 'wiki7b'
)
$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath 'C:\llama-cuda'
if ($Profile -eq 'general27b') {
    & .\llama-server.exe -hf bartowski/Qwen3.8-27B-GGUF:Q3_K_M --device CUDA0 --alias qwen38-local --host 0.0.0.0 --port 8080 --ctx-size 32768 --n-gpu-layers auto --flash-attn on --cache-type-k q8_0 --cache-type-v q8_0 --parallel 1 --jinja
} else {
    $model = Join-Path $PSScriptRoot 'models\Qwen2.5-7B-Instruct-Q5_K_M.gguf'
    if (-not (Test-Path -LiteralPath $model)) { throw "Model not downloaded: $model" }
    & .\llama-server.exe --model $model --device CUDA0 --alias wiki-qwen25-7b --host 127.0.0.1 --port 8080 --ctx-size 16384 --n-gpu-layers auto --flash-attn on --cache-type-k q8_0 --cache-type-v q8_0 --parallel 1 --jinja --reasoning off --n-predict 384
}
exit $LASTEXITCODE
