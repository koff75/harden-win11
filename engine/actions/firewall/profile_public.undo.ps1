# profile_public.undo.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) {
    $inputJson = ($input | Out-String).Trim()
} else {
    $inputJson = [Console]::In.ReadToEnd()
}

if (-not $inputJson.Trim()) {
    Write-Error "undo requires JSON input with Enabled/DefaultInboundAction/DefaultOutboundAction fields"
    exit 1
}
$state = $inputJson | ConvertFrom-Json

Set-NetFirewallProfile -Profile Public `
    -Enabled               ([string]$state.Enabled) `
    -DefaultInboundAction  ([string]$state.DefaultInboundAction) `
    -DefaultOutboundAction ([string]$state.DefaultOutboundAction)

@{ ok = $true } | ConvertTo-Json -Compress
