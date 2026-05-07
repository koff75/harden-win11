# sample_submission.test.ps1
# 0=AlwaysPrompt, 1=SendSafeSamples, 2=NeverSend, 3=SendAllSamples
# Conforme = 1 (SendSafeSamples) — équilibre privacy / sécurité.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$raw = [int](Get-MpPreference).SubmitSamplesConsent
$compliant = $raw -eq 1
$names = @{ 0 = 'AlwaysPrompt'; 1 = 'SendSafeSamples'; 2 = 'NeverSend'; 3 = 'SendAllSamples' }
$mode = if ($names.ContainsKey($raw)) { $names[$raw] } else { "Unknown($raw)" }

@{
    compliant = $compliant
    current   = @{ SubmitSamplesConsent = $mode }
} | ConvertTo-Json -Compress -Depth 10
