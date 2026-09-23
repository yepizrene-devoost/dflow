# install.ps1 - install the latest dflow release on Windows.
#
# Usage:
#   irm https://github.com/yepizrene-devoost/dflow/releases/latest/download/install.ps1 | iex
#   # or download it and run:
#   .\install.ps1
#
# Environment variables:
#   DFLOW_VERSION   Pin a version instead of installing the latest (e.g. 0.2.0 or v0.2.0).
#
# Installs to %LOCALAPPDATA%\Programs\dflow and never requires admin rights.

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Windows PowerShell 5.1 defaults to TLS 1.0, which GitHub no longer accepts.
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$RepositoryUrl = 'https://github.com/yepizrene-devoost/dflow'

# Windows PowerShell 5.1 needs -UseBasicParsing to skip the IE engine;
# PowerShell 6+ always uses basic parsing and does not need the switch.
function Get-BasicParsingSwitch {
    if ($PSVersionTable.PSVersion.Major -ge 6) {
        return @{}
    }

    return @{ UseBasicParsing = $true }
}

function Get-RemoteFile {
    param(
        [Parameter(Mandatory = $true)][string]$Uri,
        [Parameter(Mandatory = $true)][string]$Destination
    )

    $basicParsing = Get-BasicParsingSwitch
    Invoke-WebRequest -Uri $Uri -OutFile $Destination -ErrorAction Stop @basicParsing
}

function Resolve-LatestTag {
    param([Parameter(Mandatory = $true)][string]$Repository)

    $latestUrl = "$Repository/releases/latest"
    $location = $null
    $basicParsing = Get-BasicParsingSwitch

    try {
        # Do not follow the redirect: we want the tag embedded in the Location header.
        $response = Invoke-WebRequest -Uri $latestUrl -MaximumRedirection 0 -ErrorAction Stop @basicParsing
        if ($null -ne $response -and $null -ne $response.Headers['Location']) {
            $location = [string]$response.Headers['Location']
        }
    }
    catch {
        # Windows PowerShell 5.1 surfaces the redirect as a terminating error,
        # so the Location header has to come from the exception response.
        # Guard the response access: a non-redirect failure (for example a 404)
        # may not expose a Location header at all.
        try {
            $webResponse = $_.Exception.Response
            if ($null -ne $webResponse -and $null -ne $webResponse.Headers['Location']) {
                $location = [string]$webResponse.Headers['Location']
            }
        }
        catch {
            # Not a redirect response; fall through to the clear error below.
        }
    }

    if ([string]::IsNullOrWhiteSpace($location)) {
        throw "Could not resolve the latest dflow release from $latestUrl. Set `$env:DFLOW_VERSION to install a specific version."
    }

    $tag = ($location.TrimEnd('/') -split '/')[-1]
    if ([string]::IsNullOrWhiteSpace($tag)) {
        throw "Could not read a release tag from '$location'."
    }

    return $tag
}

# --- Platform detection -------------------------------------------------------

switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { $arch = 'x86_64' }
    'ARM64' { $arch = 'arm64' }
    default {
        throw "Unsupported processor architecture '$env:PROCESSOR_ARCHITECTURE'. dflow ships Windows builds for x86_64 (AMD64) and arm64 only."
    }
}

if ([string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) {
    throw 'LOCALAPPDATA is not set, so the install location cannot be determined.'
}

$installDir = Join-Path $env:LOCALAPPDATA 'Programs\dflow'

# --- Version resolution -------------------------------------------------------

$pinnedVersion = $env:DFLOW_VERSION
if (-not [string]::IsNullOrWhiteSpace($pinnedVersion)) {
    # The checksum asset uses the bare version and the download URLs the v-prefixed tag.
    $version = $pinnedVersion.Trim() -replace '^[vV]', ''
    $tag = "v$version"
    Write-Host "==> Installing dflow $version (pinned)."
}
else {
    Write-Host '==> Resolving the latest dflow release...'
    $tag = Resolve-LatestTag -Repository $RepositoryUrl
    $version = $tag -replace '^[vV]', ''
    Write-Host "==> Latest release is $tag."
}

$assetName = "dflow_Windows_$arch.zip"
$assetUrl = "$RepositoryUrl/releases/download/$tag/$assetName"
$checksumUrl = "$RepositoryUrl/releases/download/$tag/dflow_${version}_checksums.txt"

# --- Download, verify, extract ------------------------------------------------

$progressWas = $ProgressPreference
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("dflow-install-" + [Guid]::NewGuid().ToString('N'))

try {
    $ProgressPreference = 'SilentlyContinue'

    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    $archivePath = Join-Path $tempDir $assetName
    $checksumPath = Join-Path $tempDir "dflow_${version}_checksums.txt"

    Write-Host "==> Downloading $assetName..."
    Get-RemoteFile -Uri $assetUrl -Destination $archivePath

    Write-Host '==> Downloading checksums...'
    Get-RemoteFile -Uri $checksumUrl -Destination $checksumPath

    $expectedHash = $null
    foreach ($line in (Get-Content -LiteralPath $checksumPath)) {
        $parts = $line -split '\s+', 2
        if ($parts.Count -eq 2) {
            $name = $parts[1].Trim().TrimStart('*')
            if ($name -eq $assetName) {
                $expectedHash = $parts[0].Trim().ToLowerInvariant()
                break
            }
        }
    }

    if ([string]::IsNullOrWhiteSpace($expectedHash)) {
        throw "No checksum entry for $assetName was found in the checksums file."
    }

    $actualHash = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualHash -ne $expectedHash) {
        throw "Checksum mismatch for ${assetName}: expected $expectedHash but got $actualHash. Aborting without extracting."
    }
    Write-Host '==> Checksum verified.'

    $extractDir = Join-Path $tempDir 'extract'
    Expand-Archive -LiteralPath $archivePath -DestinationPath $extractDir -Force

    $extractedBinary = Join-Path $extractDir 'dflow.exe'
    if (-not (Test-Path -LiteralPath $extractedBinary)) {
        throw "The archive did not contain dflow.exe."
    }

    if (-not (Test-Path -LiteralPath $installDir)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    $targetBinary = Join-Path $installDir 'dflow.exe'
    if (Test-Path -LiteralPath $targetBinary) {
        # Replace the old copy instead of overwriting in place, so a running or
        # locked binary is surfaced as a clean failure.
        Remove-Item -LiteralPath $targetBinary -Force
    }

    Copy-Item -LiteralPath $extractedBinary -Destination $targetBinary -Force
    Write-Host "==> Installed dflow $version to $targetBinary"
}
finally {
    $ProgressPreference = $progressWas
    if (Test-Path -LiteralPath $tempDir) {
        Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# --- PATH ---------------------------------------------------------------------

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($null -eq $userPath) {
    $userPath = ''
}

$pathEntries = @($userPath -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })

$alreadyOnPath = $false
foreach ($entry in $pathEntries) {
    if ($entry.TrimEnd('\') -ieq $installDir.TrimEnd('\')) {
        $alreadyOnPath = $true
        break
    }
}

if ($alreadyOnPath) {
    Write-Host "==> $installDir is already on your user PATH."
}
else {
    $newPath = if ([string]::IsNullOrWhiteSpace($userPath)) { $installDir } else { "$userPath;$installDir" }
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    Write-Host "==> Added $installDir to your user PATH."
    Write-Host '    Open a new terminal (or sign out and back in) for the change to take effect.'
}

Write-Host ''
Write-Host "dflow $version is installed. Run 'dflow --help' to get started."
