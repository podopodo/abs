param(
  [string]$Repository = "podopodo/abs",
  [string]$Version = "latest",
  [string]$InstallDir = "$HOME\bin"
)
$ErrorActionPreference = "Stop"
$arch = if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq "Arm64") { "arm64" } else { "amd64" }
$asset = "acp_windows_${arch}.zip"
$base = if ($Version -eq "latest") { "https://github.com/$Repository/releases/latest/download" } else { "https://github.com/$Repository/releases/download/$Version" }
$temp = Join-Path ([System.IO.Path]::GetTempPath()) ("acp-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $temp | Out-Null
try {
  Invoke-WebRequest "$base/$asset" -OutFile (Join-Path $temp $asset)
  Invoke-WebRequest "$base/checksums.txt" -OutFile (Join-Path $temp "checksums.txt")
  $line = Select-String -Path (Join-Path $temp "checksums.txt") -Pattern "  $([regex]::Escape($asset))$" | Select-Object -First 1
  if (-not $line) { throw "Checksum entry not found" }
  $expected = ($line.Line -split '\s+')[0].ToLowerInvariant()
  $actual = (Get-FileHash (Join-Path $temp $asset) -Algorithm SHA256).Hash.ToLowerInvariant()
  if ($expected -ne $actual) { throw "Checksum mismatch" }
  Expand-Archive (Join-Path $temp $asset) -DestinationPath $temp -Force
  New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
  Copy-Item (Join-Path $temp "acp.exe") (Join-Path $InstallDir "acp.exe") -Force
  Write-Host "Installed $(Join-Path $InstallDir 'acp.exe')"
  & (Join-Path $InstallDir "acp.exe") version
} finally {
  Remove-Item $temp -Recurse -Force -ErrorAction SilentlyContinue
}
