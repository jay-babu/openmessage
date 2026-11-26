
$ErrorActionPreference = 'Stop'
$hostManifest = Join-Path $PSScriptRoot 'openmessage-browser-host.json'
$hostExecutable = Join-Path $PSScriptRoot 'openmessage-browser-host.exe'
$extension = Join-Path $PSScriptRoot 'browser-extension'
if (-not (Test-Path -LiteralPath $hostManifest -PathType Leaf) -or
    -not (Test-Path -LiteralPath $hostExecutable -PathType Leaf) -or
    -not (Test-Path -LiteralPath (Join-Path $extension 'manifest.json') -PathType Leaf)) {
    throw 'Extract the full Windows archive before registering browser refresh.'
}

$registryKey = 'HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\com.openmessage.google_refresh'
New-Item -Path $registryKey -Force | Out-Null
Set-Item -LiteralPath $registryKey -Value $hostManifest
Write-Host 'Registered the OpenMessage native host for the current Windows user.'
Write-Host "In Edge, open edge://extensions and use Load unpacked to select $extension"
