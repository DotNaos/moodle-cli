$ErrorActionPreference = "Stop"

$Owner = "DotNaos"
$Repo = "moodle-cli"
$Version = if ($env:VERSION) { $env:VERSION } else { "latest" }
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\moodle-cli\bin" }
$ChecksumFile = "checksums.txt"

function Resolve-AssetName {
  switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { return "moodle_windows_amd64.zip" }
    "ARM64" { return "moodle_windows_arm64.zip" }
    default { throw "Unsupported Windows architecture: $env:PROCESSOR_ARCHITECTURE" }
  }
}

function Get-BaseUrl {
  if ($Version -eq "latest") {
    return "https://github.com/$Owner/$Repo/releases/latest/download"
  }
  return "https://github.com/$Owner/$Repo/releases/download/$Version"
}

function Get-ExpectedChecksum {
  param(
    [string]$ChecksumPath,
    [string]$AssetName
  )

  $line = Select-String -Path $ChecksumPath -Pattern " $([regex]::Escape($AssetName))$" | Select-Object -First 1
  if (-not $line) {
    throw "Could not find checksum for $AssetName."
  }

  return ($line.Line -split '\s+')[0]
}

$AssetName = Resolve-AssetName
$BaseUrl = Get-BaseUrl
$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("moodle-install-" + [System.Guid]::NewGuid().ToString("n"))

New-Item -ItemType Directory -Path $TempDir | Out-Null
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

$ZipPath = Join-Path $TempDir $AssetName
$ChecksumPath = Join-Path $TempDir $ChecksumFile
$ExtractDir = Join-Path $TempDir "extract"

try {
  Invoke-WebRequest -Uri "$BaseUrl/$AssetName" -OutFile $ZipPath
  Invoke-WebRequest -Uri "$BaseUrl/$ChecksumFile" -OutFile $ChecksumPath

  $Expected = Get-ExpectedChecksum -ChecksumPath $ChecksumPath -AssetName $AssetName
  $Actual = (Get-FileHash -Path $ZipPath -Algorithm SHA256).Hash.ToLowerInvariant()
  if ($Expected.ToLowerInvariant() -ne $Actual) {
    throw "Checksum verification failed for $AssetName."
  }

  Expand-Archive -Path $ZipPath -DestinationPath $ExtractDir -Force
  Copy-Item -Path (Join-Path $ExtractDir "moodle.exe") -Destination (Join-Path $InstallDir "moodle.exe") -Force

  Write-Host "Installed moodle to $(Join-Path $InstallDir 'moodle.exe')"
  if (-not (($env:PATH -split ';') -contains $InstallDir)) {
    Write-Warning "Add $InstallDir to your PATH if it is not already there."
  }
}
finally {
  Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
}
