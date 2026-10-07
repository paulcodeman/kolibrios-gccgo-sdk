param(
    [string]$SdkRoot = (Join-Path $env:USERPROFILE 'Desktop/kolibrios/kolibrios-gccgo-sdk')
)

$ErrorActionPreference = 'Stop'
$healthUri = 'http://127.0.0.1:18081/health'
try {
    $health = Invoke-RestMethod -Uri $healthUri -TimeoutSec 3
    if ($health.status -eq 'ok') {
        Write-Output 'Local Qwen is already ready on port 18081.'
        return
    }
} catch {
    # Start the existing model below when the endpoint is not running.
}

$resolvedSdk = (Resolve-Path -LiteralPath $SdkRoot).Path
$serverRelative = '.build-cache/opencode/real-ai-server/llama-b11435/llama-b11435/llama-server'
$modelRelative = '.build-cache/opencode/real-ai-server/qwen2.5-0.5b-instruct-q4_k_m.gguf'
foreach ($relative in @($serverRelative, $modelRelative)) {
    if (-not (Test-Path -LiteralPath (Join-Path $resolvedSdk $relative) -PathType Leaf)) {
        throw "The cached local server or model is missing: $relative. Pass -SdkRoot with the SDK directory."
    }
}
$linuxSdk = (& wsl.exe --exec wslpath -a -u $resolvedSdk).Trim()
if ($LASTEXITCODE -ne 0 -or -not $linuxSdk) {
    throw 'WSL could not resolve the SDK directory.'
}
$serverPath = $linuxSdk + '/' + $serverRelative
$modelPath = $linuxSdk + '/' + $modelRelative
$serverArguments = @('--exec', ('"' + $serverPath + '"'), '-m', ('"' + $modelPath + '"'),
    '--host', '127.0.0.1', '--port', '18081', '--alias', 'kolibri-test-model',
    '-c', '32768', '-t', '2', '-np', '1', '--n-predict', '128',
    '--no-webui', '--jinja', '--sse-ping-interval', '-1')
Start-Process -FilePath 'wsl.exe' -ArgumentList $serverArguments -WindowStyle Hidden
for ($attempt = 0; $attempt -lt 60; $attempt++) {
    try {
        $health = Invoke-RestMethod -Uri $healthUri -TimeoutSec 2
        if ($health.status -eq 'ok') {
            Write-Output 'Local Qwen is ready on port 18081. You can start OpenCode in QEMU.'
            return
        }
    } catch {
        # Loading the model can take several seconds.
    }
    Start-Sleep -Seconds 1
}
throw 'The local model did not become ready. Check the WSL server and model files.'
