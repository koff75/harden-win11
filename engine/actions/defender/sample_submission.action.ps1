# sample_submission.action.ps1
# Configure SubmitSamplesConsent à SendSafeSamples.
# 0=AlwaysPrompt, 1=SendSafeSamples, 2=NeverSend, 3=SendAllSamples

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$names = @{ 0 = 'AlwaysPrompt'; 1 = 'SendSafeSamples'; 2 = 'NeverSend'; 3 = 'SendAllSamples' }

$rawBefore = [int](Get-MpPreference).SubmitSamplesConsent
$before = @{
    SubmitSamplesConsent = if ($names.ContainsKey($rawBefore)) { $names[$rawBefore] } else { "Unknown($rawBefore)" }
}

Set-MpPreference -SubmitSamplesConsent SendSafeSamples

$rawAfter = [int](Get-MpPreference).SubmitSamplesConsent
$after = @{
    SubmitSamplesConsent = if ($names.ContainsKey($rawAfter)) { $names[$rawAfter] } else { "Unknown($rawAfter)" }
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
