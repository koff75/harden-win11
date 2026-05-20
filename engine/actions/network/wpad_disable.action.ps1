# wpad_disable.action.ps1
# WPAD desactive (anti-poisoning).
#
# IMPORTANT : on met Start=3 (Manual) et NON Start=4 (Disabled).
# Raison : sur certaines configs Windows (notamment Win11 ARM Home), le
# service WLAN AutoConfig (WlanSvc) a une dependance implicite sur
# WinHttpAutoProxySvc. Mettre le service en Disabled empeche WlanSvc de
# demarrer au boot → WiFi casse. Avec Start=3 (Manual), le service ne
# tourne pas en arriere-plan (donc plus de broadcast WPAD vulnerable au
# poisoning), mais peut etre demarre a la demande par les dependants.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\WinHttpAutoProxySvc' `
    -Name 'Start' `
    -Value 3 `
    -Type DWord