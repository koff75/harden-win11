# bing_weather.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\appx.psm1') -Force

Invoke-AppxTest -Pattern '*Microsoft.BingWeather*'