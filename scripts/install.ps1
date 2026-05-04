[CmdletBinding()]
param(
    [string]$Version,
    [switch]$Uninstall,
    [switch]$Yes,
    [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RepoOwner = if ($env:SLOPSHELL_REPO_OWNER) { $env:SLOPSHELL_REPO_OWNER } else { "sloppy-org" }
$RepoName = if ($env:SLOPSHELL_REPO_NAME) { $env:SLOPSHELL_REPO_NAME } else { "slopshell" }
$ReleaseApiBase = if ($env:SLOPSHELL_RELEASE_API_BASE) { $env:SLOPSHELL_RELEASE_API_BASE } else { "https://api.github.com/repos/$RepoOwner/$RepoName/releases" }
$SkipBrowser = $env:SLOPSHELL_INSTALL_SKIP_BROWSER -eq "1"
$AssumeYes = $Yes.IsPresent -or ($env:SLOPSHELL_ASSUME_YES -eq "1")

$InstallRoot = if ($env:SLOPSHELL_INSTALL_ROOT) { $env:SLOPSHELL_INSTALL_ROOT } else { Join-Path $env:LOCALAPPDATA "slopshell" }
$BinaryPath = Join-Path $InstallRoot "slopshell.exe"
$DataRoot = Join-Path $InstallRoot "data"
$ProjectDir = Join-Path $DataRoot "project"
$WebDataDir = Join-Path $DataRoot "web-data"
$PiperRoot = Join-Path $DataRoot "piper-tts"
$PiperVenv = Join-Path $PiperRoot "venv"
$ModelDir = Join-Path $PiperRoot "models"
$ScriptDir = Join-Path $DataRoot "scripts"
$PiperScriptPath = Join-Path $ScriptDir "piper_tts_server.py"
$LlmEnvDir = Join-Path $env:USERPROFILE ".config\slopshell"
$LlmEnvFile = Join-Path $LlmEnvDir "llm.env"
$WebLauncherPath = Join-Path $ScriptDir "start-slopshell-web.ps1"
$WebHost = if ($env:SLOPSHELL_WEB_HOST) { $env:SLOPSHELL_WEB_HOST } else { "127.0.0.1" }

function Write-Log {
    param([string]$Message)
    Write-Host "[slopshell-install] $Message"
}

function Throw-InstallError {
    param([string]$Message)
    throw "[slopshell-install] ERROR: $Message"
}

function Invoke-Step {
    param([ScriptBlock]$Action, [string]$Display)
    if ($DryRun.IsPresent) {
        Write-Log "[dry-run] $Display"
        return
    }
    & $Action
}

function Confirm-DefaultYes {
    param([string]$Prompt)
    if ($AssumeYes) {
        Write-Log "SLOPSHELL_ASSUME_YES accepted: $Prompt"
        return $true
    }
    if (-not [Environment]::UserInteractive) {
        Write-Log "non-interactive session defaults to yes: $Prompt"
        return $true
    }
    $answer = Read-Host "$Prompt [Y/n]"
    return [string]::IsNullOrWhiteSpace($answer) -or ($answer -match '^(?i)y(es)?$')
}

function Normalize-Version {
    param([string]$Raw)
    if (-not $Raw) { return "" }
    $clean = $Raw.TrimStart('v', 'V')
    return "v$clean"
}

function Get-DefaultOpenAIBaseUrl {
    if ($env:SLOPSHELL_DEFAULT_OPENAI_BASE_URL) {
        return $env:SLOPSHELL_DEFAULT_OPENAI_BASE_URL
    }
    return "http://127.0.0.1:8080/v1"
}

function Load-LlmEnvFile {
    param([string]$Path)
    $values = @{}
    if (-not (Test-Path $Path)) {
        return $values
    }
    foreach ($line in Get-Content -Path $Path) {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        $trimmed = $line.Trim()
        if ($trimmed.StartsWith("#")) { continue }
        $parts = $trimmed.Split('=', 2)
        if ($parts.Count -eq 2) {
            $values[$parts[0]] = $parts[1]
        }
    }
    return $values
}

function Write-LlmEnvFile {
    $defaultBaseUrl = Get-DefaultOpenAIBaseUrl
    $defaultIntentUrl = $defaultBaseUrl -replace '/v1$',''
    $intentUrl = if ($env:SLOPSHELL_INTENT_LLM_URL) { $env:SLOPSHELL_INTENT_LLM_URL } else { $defaultIntentUrl }
    $intentModel = if ($env:SLOPSHELL_INTENT_LLM_MODEL) { $env:SLOPSHELL_INTENT_LLM_MODEL } else { "qwen" }
    $intentProfile = if ($env:SLOPSHELL_INTENT_LLM_PROFILE) { $env:SLOPSHELL_INTENT_LLM_PROFILE } else { "default" }
    $intentProfileOptions = if ($env:SLOPSHELL_INTENT_LLM_PROFILE_OPTIONS) { $env:SLOPSHELL_INTENT_LLM_PROFILE_OPTIONS } else { $intentProfile }
    $assistantUrl = if ($env:SLOPSHELL_ASSISTANT_LLM_URL) { $env:SLOPSHELL_ASSISTANT_LLM_URL } else { $intentUrl }
    $assistantModel = if ($env:SLOPSHELL_ASSISTANT_LLM_MODEL) { $env:SLOPSHELL_ASSISTANT_LLM_MODEL } else { $intentModel }
    $codexBaseUrl = if ($env:SLOPSHELL_CODEX_BASE_URL) { $env:SLOPSHELL_CODEX_BASE_URL } else { $defaultBaseUrl }
    $codexFastModel = if ($env:SLOPSHELL_CODEX_FAST_MODEL) { $env:SLOPSHELL_CODEX_FAST_MODEL } else { $intentModel }
    $codexLocalModel = if ($env:SLOPSHELL_CODEX_LOCAL_MODEL) { $env:SLOPSHELL_CODEX_LOCAL_MODEL } else { $intentModel }

    $content = @(
        "SLOPSHELL_INTENT_LLM_URL=$intentUrl"
        "SLOPSHELL_INTENT_LLM_MODEL=$intentModel"
        "SLOPSHELL_INTENT_LLM_PROFILE=$intentProfile"
        "SLOPSHELL_INTENT_LLM_PROFILE_OPTIONS=$intentProfileOptions"
        "SLOPSHELL_ASSISTANT_LLM_URL=$assistantUrl"
        "SLOPSHELL_ASSISTANT_LLM_MODEL=$assistantModel"
        "SLOPSHELL_CODEX_BASE_URL=$codexBaseUrl"
        "SLOPSHELL_CODEX_FAST_MODEL=$codexFastModel"
        "SLOPSHELL_CODEX_LOCAL_MODEL=$codexLocalModel"
    )

    if ($DryRun.IsPresent) {
        Write-Log "[dry-run] Write $LlmEnvFile"
        return [ordered]@{
            SLOPSHELL_INTENT_LLM_URL = $intentUrl
            SLOPSHELL_INTENT_LLM_MODEL = $intentModel
            SLOPSHELL_CODEX_BASE_URL = $codexBaseUrl
        }
    }

    New-Item -ItemType Directory -Force -Path $LlmEnvDir | Out-Null
    Set-Content -Path $LlmEnvFile -Value $content
    return [ordered]@{
        SLOPSHELL_INTENT_LLM_URL = $intentUrl
        SLOPSHELL_INTENT_LLM_MODEL = $intentModel
        SLOPSHELL_CODEX_BASE_URL = $codexBaseUrl
    }
}

function Initialize-LlmEnvFile {
    if ((Test-Path $LlmEnvFile) -and -not $env:SLOPSHELL_INTENT_LLM_URL -and -not $env:SLOPSHELL_INTENT_LLM_MODEL -and -not $env:SLOPSHELL_CODEX_BASE_URL -and -not $env:SLOPSHELL_CODEX_FAST_MODEL -and -not $env:SLOPSHELL_CODEX_LOCAL_MODEL) {
        Write-Log "using existing OpenAI-compatible LLM config from $LlmEnvFile"
        return (Load-LlmEnvFile -Path $LlmEnvFile)
    }
    Write-Log "writing OpenAI-compatible LLM config to $LlmEnvFile"
    return (Write-LlmEnvFile)
}

function Resolve-Arch {
    if ($env:PROCESSOR_ARCHITECTURE -match 'ARM64') { return "arm64" }
    return "amd64"
}

function Require-Codex {
    $cmd = Get-Command codex -ErrorAction SilentlyContinue
    if (-not $cmd) {
        Throw-InstallError "codex app-server is required but codex is not in PATH"
    }
    return $cmd.Source
}

function Require-Python {
    $python = Get-Command python -ErrorAction SilentlyContinue
    if (-not $python) {
        $python = Get-Command py -ErrorAction SilentlyContinue
    }
    if (-not $python) {
        Throw-InstallError "Python 3.10+ is required"
    }

    $versionOutput = & $python.Source -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')"
    if (-not $versionOutput) {
        Throw-InstallError "unable to detect Python version"
    }
    $parts = $versionOutput.Trim().Split('.')
    $major = [int]$parts[0]
    $minor = [int]$parts[1]
    if ($major -lt 3 -or ($major -eq 3 -and $minor -lt 10)) {
        Throw-InstallError "Python 3.10+ is required"
    }
    return $python.Source
}

function Get-Release {
    param([string]$Requested)

    if ($env:SLOPSHELL_RELEASE_JSON) {
        return ($env:SLOPSHELL_RELEASE_JSON | ConvertFrom-Json)
    }
    if ($DryRun.IsPresent) {
        $tag = if ($Requested) { Normalize-Version $Requested } else { "v0.0.0-test" }
        $plain = $tag.TrimStart('v')
        $arch = Resolve-Arch
        $json = @"
{
  "tag_name": "$tag",
  "assets": [
    {"name":"slopshell_${plain}_windows_${arch}.zip","browser_download_url":"https://example.invalid/slopshell.zip"},
    {"name":"checksums.txt","browser_download_url":"https://example.invalid/checksums.txt"}
  ]
}
"@
        return ($json | ConvertFrom-Json)
    }

    $url = if ($Requested) {
        "$ReleaseApiBase/tags/$(Normalize-Version $Requested)"
    } else {
        "$ReleaseApiBase/latest"
    }
    return Invoke-RestMethod -Uri $url -Headers @{"Accept"="application/vnd.github+json"}
}

function Get-Asset {
    param($Release, [string]$Name)
    $asset = $Release.assets | Where-Object { $_.name -eq $Name } | Select-Object -First 1
    if (-not $asset) {
        Throw-InstallError "release missing asset $Name"
    }
    return $asset
}

function Ensure-InstallDirectories {
    Invoke-Step -Display "Create install directories" -Action {
        New-Item -ItemType Directory -Force -Path $InstallRoot, $DataRoot, $ProjectDir, $WebDataDir, $PiperRoot, $ModelDir, $ScriptDir, $LlmEnvDir | Out-Null
    }
}

function Install-Binary {
    param($Release)

    $tag = $Release.tag_name
    if (-not $tag) { Throw-InstallError "release did not provide tag_name" }
    $plainVersion = $tag.TrimStart('v')
    $arch = Resolve-Arch
    $assetName = "slopshell_${plainVersion}_windows_${arch}.zip"
    $asset = Get-Asset -Release $Release -Name $assetName

    if ($DryRun.IsPresent) {
        Invoke-Step -Display "Install slopshell.exe to $BinaryPath" -Action {}
        return $tag
    }

    $checksumAsset = Get-Asset -Release $Release -Name "checksums.txt"
    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("slopshell-install-" + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $tmpDir | Out-Null
    try {
        $zipPath = Join-Path $tmpDir $assetName
        $checksumPath = Join-Path $tmpDir "checksums.txt"
        Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zipPath
        Invoke-WebRequest -Uri $checksumAsset.browser_download_url -OutFile $checksumPath

        $expected = Select-String -Path $checksumPath -Pattern ("\s$([regex]::Escape($assetName))$") | ForEach-Object { ($_ -split '\s+')[0] } | Select-Object -First 1
        if (-not $expected) {
            Throw-InstallError "checksum entry missing for $assetName"
        }
        $actual = (Get-FileHash -Algorithm SHA256 -Path $zipPath).Hash.ToLowerInvariant()
        if ($actual -ne $expected.ToLowerInvariant()) {
            Throw-InstallError "checksum mismatch for $assetName"
        }

        $extractDir = Join-Path $tmpDir "extract"
        Expand-Archive -Path $zipPath -DestinationPath $extractDir -Force
        $exeSource = Get-ChildItem -Path $extractDir -Filter "slopshell.exe" -Recurse | Select-Object -First 1
        if (-not $exeSource) {
            Throw-InstallError "slopshell.exe missing in release archive"
        }
        Copy-Item -Path $exeSource.FullName -Destination $BinaryPath -Force

        $piperSource = Get-ChildItem -Path $extractDir -Filter "piper_tts_server.py" -Recurse | Select-Object -First 1
        if (-not $piperSource) {
            Throw-InstallError "scripts/piper_tts_server.py missing in release archive"
        }
        Copy-Item -Path $piperSource.FullName -Destination $PiperScriptPath -Force
    }
    finally {
        Remove-Item -Recurse -Force -Path $tmpDir -ErrorAction SilentlyContinue
    }

    return $tag
}

function Ensure-UserPath {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $parts = @()
    if ($userPath) {
        $parts = $userPath.Split(';', [System.StringSplitOptions]::RemoveEmptyEntries)
    }
    if ($parts -contains $InstallRoot) {
        return
    }
    if ($DryRun.IsPresent) {
        Write-Log "[dry-run] Add $InstallRoot to user PATH"
        return
    }
    $newPath = (($parts + $InstallRoot) -join ';')
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Log "added $InstallRoot to user PATH"
}

function Setup-Piper {
    Write-Host "=== Piper TTS (GPL, runs as HTTP sidecar) ==="
    Write-Host "Piper TTS will be installed as a local HTTP service."
    Write-Host "License: GPL (isolated via HTTP boundary, does not affect Slopshell MIT license)"
    Write-Host "Voice models: en_GB-alan-medium (MIT-compatible)"

    if (-not (Confirm-DefaultYes "Install Piper TTS?")) {
        Write-Log "skipping Piper TTS setup"
        return
    }

    if ($DryRun.IsPresent) {
        Invoke-Step -Display "Create Piper venv and install piper-tts fastapi uvicorn" -Action {}
        Invoke-Step -Display "Download Piper voice model en_GB-alan-medium" -Action {}
        return
    }

    $python = Require-Python
    if (-not (Test-Path $PiperVenv)) {
        & $python -m venv $PiperVenv
    }
    $venvPython = Join-Path $PiperVenv "Scripts\python.exe"
    & $venvPython -m pip install --upgrade pip
    & $venvPython -m pip install piper-tts fastapi uvicorn

    $onnx = Join-Path $ModelDir "en_GB-alan-medium.onnx"
    $json = Join-Path $ModelDir "en_GB-alan-medium.onnx.json"
    if (-not (Test-Path $onnx)) {
        Write-Log "model card: https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_GB/alan/medium/MODEL_CARD"
        Invoke-WebRequest -Uri "https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_GB/alan/medium/en_GB-alan-medium.onnx" -OutFile $onnx
        Invoke-WebRequest -Uri "https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_GB/alan/medium/en_GB-alan-medium.onnx.json" -OutFile $json
    }
}

function Write-TaskFiles {
    param([string]$CodexPath)

    $webScript = @"
\$envFile = '$LlmEnvFile'
if (Test-Path \$envFile) {
    foreach (\$line in Get-Content -Path \$envFile) {
        if ([string]::IsNullOrWhiteSpace(\$line)) { continue }
        \$trimmed = \$line.Trim()
        if (\$trimmed.StartsWith('#')) { continue }
        \$parts = \$trimmed.Split('=', 2)
        if (\$parts.Count -eq 2) {
            [Environment]::SetEnvironmentVariable(\$parts[0], \$parts[1])
        }
    }
}
[Environment]::SetEnvironmentVariable('SLOPSHELL_BRAIN_GTD_SYNC', 'on')
& '$BinaryPath' server --workspace-dir '$ProjectDir' --data-dir '$WebDataDir' --web-host $WebHost --web-port 8420 --app-server-url ws://127.0.0.1:8787 --tts-url http://127.0.0.1:8424
"@
    $piperCmd = 'set "PIPER_MODEL_DIR=' + $ModelDir + '" && "' + (Join-Path $PiperVenv 'Scripts\python.exe') + '" -m uvicorn piper_tts_server:app --app-dir "' + $ScriptDir + '" --host 127.0.0.1 --port 8424'
    $codexCmd = '"' + $CodexPath + '" app-server --listen ws://127.0.0.1:8787'
    $webCmd = 'powershell.exe -NoProfile -ExecutionPolicy Bypass -File "' + $WebLauncherPath + '"'

    if ($DryRun.IsPresent) {
        Write-Log "[dry-run] Register scheduled tasks slopshell-web, slopshell-piper-tts, slopshell-codex-app-server"
        return
    }

    Set-Content -Path $WebLauncherPath -Value $webScript
    schtasks /Create /SC ONLOGON /TN "slopshell-codex-app-server" /TR $codexCmd /F | Out-Null
    schtasks /Create /SC ONLOGON /TN "slopshell-piper-tts" /TR ("cmd /c " + $piperCmd) /F | Out-Null
    schtasks /Run /TN "slopshell-codex-app-server" | Out-Null
    schtasks /Run /TN "slopshell-piper-tts" | Out-Null
    schtasks /Create /SC ONLOGON /TN "slopshell-web" /TR $webCmd /F | Out-Null
    schtasks /Run /TN "slopshell-web" | Out-Null
    schtasks /Delete /TN "slopshell-llm" /F | Out-Null 2>$null
    schtasks /Delete /TN "slopshell-codex-llm" /F | Out-Null 2>$null
}

function Print-WindowsSTTNotice {
    Write-Log "Speech-to-text requires voxtype (Linux/macOS only)"
}

function Open-Browser {
    if ($SkipBrowser) {
        Write-Log "skipping browser open due to SLOPSHELL_INSTALL_SKIP_BROWSER=1"
        return
    }
    if ($DryRun.IsPresent) {
        Write-Log "[dry-run] Start browser at http://127.0.0.1:8420"
        return
    }
    Start-Process "http://127.0.0.1:8420" | Out-Null
}

function Print-Summary {
    param([string]$Tag, $LlmConfig)
    Write-Host ""
    Write-Host "Install complete"
    Write-Host "  Version:      $Tag"
    Write-Host "  Binary:       $BinaryPath"
    Write-Host "  Data root:    $DataRoot"
    Write-Host "  Project dir:  $ProjectDir"
    Write-Host "  Piper models: $ModelDir"
    Write-Host "  Web URL:      http://127.0.0.1:8420"
    Write-Host "  LLM env file: $LlmEnvFile"
    Write-Host "  Intent LLM:   $($LlmConfig['SLOPSHELL_INTENT_LLM_URL'])"
    Write-Host "  Codex LLM:    $($LlmConfig['SLOPSHELL_CODEX_BASE_URL'])"
}

function Remove-Task {
    param([string]$TaskName)
    if ($DryRun.IsPresent) {
        Write-Log "[dry-run] Delete scheduled task $TaskName"
        return
    }
    schtasks /Delete /TN $TaskName /F | Out-Null 2>$null
}

function Uninstall-Slopshell {
    Remove-Task "slopshell-web"
    Remove-Task "slopshell-llm"
    Remove-Task "slopshell-codex-llm"
    Remove-Task "slopshell-piper-tts"
    Remove-Task "slopshell-codex-app-server"

    if ($DryRun.IsPresent) {
        Write-Log "[dry-run] Remove $BinaryPath"
    } else {
        Remove-Item -Force -ErrorAction SilentlyContinue -Path $BinaryPath
    }

    if (Confirm-DefaultYes "Remove $DataRoot data directory?") {
        if ($DryRun.IsPresent) {
            Write-Log "[dry-run] Remove $DataRoot"
        } else {
            Remove-Item -Recurse -Force -ErrorAction SilentlyContinue -Path $DataRoot
        }
    }

    Write-Log "uninstall complete"
}

function Install-Slopshell {
    $codexPath = Require-Codex
    Require-Python | Out-Null
    Ensure-InstallDirectories
    $release = Get-Release -Requested $Version
    $tag = Install-Binary -Release $release
    Ensure-UserPath
    Setup-Piper
    $llmConfig = Initialize-LlmEnvFile
    Print-WindowsSTTNotice
    Write-TaskFiles -CodexPath $codexPath
    Open-Browser
    Print-Summary -Tag $tag -LlmConfig $llmConfig
}

if ($Uninstall.IsPresent) {
    Uninstall-Slopshell
} else {
    Install-Slopshell
}
