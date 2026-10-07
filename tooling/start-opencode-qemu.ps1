param(
    [string]$RuntimeDirectory = (Join-Path $env:USERPROFILE 'Desktop/opencode-qemu'),
    [string]$SdkRoot = (Join-Path $env:USERPROFILE 'Desktop/kolibrios/kolibrios-gccgo-sdk'),
    [switch]$CheckOnly
)

$ErrorActionPreference = 'Stop'
$runtime = (Resolve-Path -LiteralPath $RuntimeDirectory).Path
$qemu = Join-Path $runtime 'qemu/qemu-system-i386.exe'
$floppy = Join-Path $runtime 'opencode-boot.img'
$disk = Join-Path $runtime 'opencode-runtime.img'
foreach ($file in @($qemu, $floppy, $disk)) {
    if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
        throw "Missing OpenCode QEMU runtime file: $file"
    }
}
$arguments = @('-machine', 'pc', '-accel', 'tcg', '-cpu', 'max', '-m', '512',
    '-vga', 'std', '-display', 'sdl', '-boot', 'order=a',
    '-drive', ('file=' + $floppy + ',format=raw,if=floppy,index=0'),
    '-drive', ('file=' + $disk + ',format=raw,if=ide,index=0'),
    '-netdev', 'user,id=net0', '-device', 'rtl8139,netdev=net0',
    '-name', 'OpenCode - ColibriOS')
if ($CheckOnly) {
    & $qemu --version
    if ($LASTEXITCODE -ne 0) { throw 'QEMU did not start.' }
    Write-Output ('CPU: max; memory: 512 MiB; network: RTL8139 / user NAT')
    Write-Output ('Boot: ' + $floppy)
    Write-Output ('Persistent disk: ' + $disk)
    return
}
& (Join-Path $PSScriptRoot 'start-opencode-local-ai.ps1') -SdkRoot $SdkRoot
& $qemu @arguments
if ($LASTEXITCODE -ne 0) { throw "QEMU exited with code $LASTEXITCODE" }
