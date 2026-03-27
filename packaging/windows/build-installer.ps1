$ErrorActionPreference = "Stop"

param(
  [Parameter(Mandatory = $true)]
  [string]$ArchivePath,
  [Parameter(Mandatory = $true)]
  [string]$Version,
  [Parameter(Mandatory = $true)]
  [ValidateSet("amd64", "arm64")]
  [string]$Arch,
  [Parameter(Mandatory = $true)]
  [string]$OutputDir
)

$RepoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$ExtractDir = Join-Path $env:TEMP ("moodle-installer-" + [System.Guid]::NewGuid().ToString("n"))
$IsccCandidates = @(
  (Join-Path ${env:ProgramFiles(x86)} "Inno Setup 6\ISCC.exe"),
  (Join-Path $env:ProgramFiles "Inno Setup 6\ISCC.exe")
)

New-Item -ItemType Directory -Force -Path $ExtractDir | Out-Null
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

try {
  Expand-Archive -Path $ArchivePath -DestinationPath $ExtractDir -Force

  $Iscc = $IsccCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1
  if (-not $Iscc) {
    $Command = Get-Command ISCC.exe -ErrorAction SilentlyContinue
    if ($Command) {
      $Iscc = $Command.Source
    }
  }
  if (-not $Iscc) {
    throw "ISCC.exe not found after installing Inno Setup."
  }

  & $Iscc `
    "/DAppVersion=$Version" `
    "/DSourceDir=$ExtractDir" `
    "/DOutputDir=$(Resolve-Path $OutputDir)" `
    "/DInstallerArch=$Arch" `
    (Join-Path $RepoRoot "packaging\windows\moodle.iss")
}
finally {
  Remove-Item -Path $ExtractDir -Recurse -Force -ErrorAction SilentlyContinue
}
