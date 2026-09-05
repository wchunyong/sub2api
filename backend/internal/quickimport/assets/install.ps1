param(
  [Parameter(Mandatory=$true)][ValidateSet('install','clean')][string]$Action,
  [Parameter(Mandatory=$true)][ValidateSet('claude','codex','opencode')][string]$Agent,
  [Parameter(Mandatory=$true)][string]$Server,
  [string]$Ticket
)
$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$uri = [Uri]$Server
if ($uri.Scheme -ne 'https' -or $uri.UserInfo -or $uri.Query -or $uri.Fragment) { throw 'HTTPS server required.' }
if ($Action -eq 'install' -and $Ticket -notmatch '^[a-f0-9]{64}$') { throw 'Generate a new import command.' }
$architecture = $env:PROCESSOR_ARCHITEW6432
if (-not $architecture) { $architecture = $env:PROCESSOR_ARCHITECTURE }
switch ($architecture.ToUpperInvariant()) {
  'AMD64' { $target = 'amd64' }
  'ARM64' { $target = 'arm64' }
  default { throw 'This Windows architecture is unsupported. Use manual configuration.' }
}
function Protect-Directory([string]$Path) {
  if (Test-Path -LiteralPath $Path) {
    if ((Get-Item -LiteralPath $Path -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Linked recovery directories are not supported.' }
  } else { New-Item -ItemType Directory -Path $Path | Out-Null }
  $sid = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
  & icacls $Path /inheritance:r /grant:r "*${sid}:(OI)(CI)F" | Out-Null
  if ($LASTEXITCODE -ne 0) { throw 'Cannot protect local recovery files.' }
}
if ((Get-Item -LiteralPath $env:USERPROFILE -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Linked user directories are not supported.' }
$recovery = Join-Path $env:USERPROFILE '.sub2api-quick-import'
Protect-Directory $recovery
$binaries = Join-Path $recovery 'bin'
Protect-Directory $binaries
$agentFolder = Join-Path $recovery $Agent
Protect-Directory $agentFolder
$download = Join-Path $binaries ([IO.Path]::GetRandomFileName() + '.download')
$restoreTemp = Join-Path $agentFolder ([IO.Path]::GetRandomFileName() + '.tmp')
try {
  $asset = $Server.TrimEnd('/') + '/api/v1/quick-import/assets/quick-import-windows-' + $target + '.exe'
  $headers = @{ 'User-Agent' = 'lianjieai-quick-import/2.0' }
  $checksum = (Invoke-WebRequest -UseBasicParsing -MaximumRedirection 0 -TimeoutSec 30 -Headers $headers -Uri ($asset + '.sha256')).Content.Trim().ToLowerInvariant()
  if ($checksum -notmatch '^[a-f0-9]{64}$') { throw 'Invalid import helper checksum.' }
  $binary = Join-Path $binaries ($checksum + '.exe')
  if (Test-Path -LiteralPath $binary) {
    if ((Get-Item -LiteralPath $binary -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Linked import helper refused.' }
  } else {
    Invoke-WebRequest -UseBasicParsing -MaximumRedirection 0 -TimeoutSec 120 -Headers $headers -Uri $asset -OutFile $download
    if ((Get-FileHash -Algorithm SHA256 -LiteralPath $download).Hash.ToLowerInvariant() -ne $checksum) { throw 'Import helper checksum mismatch. Retry with a new command.' }
    Move-Item -LiteralPath $download -Destination $binary -ErrorAction Stop
  }
  if ((Get-FileHash -Algorithm SHA256 -LiteralPath $binary).Hash.ToLowerInvariant() -ne $checksum) { throw 'Cached import helper checksum mismatch.' }
  $restore = Join-Path $agentFolder 'restore.ps1'
  if ((Test-Path -LiteralPath $restore) -and ((Get-Item -LiteralPath $restore -Force).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Linked recovery script refused.' }
  $text = "`$ErrorActionPreference = 'Stop'`r`n& (Join-Path `$PSScriptRoot '../bin/$checksum.exe') clean --agent '$Agent'`r`nif (`$LASTEXITCODE -ne 0) { throw 'Cleanup did not complete. Review the message above.' }`r`n"
  [IO.File]::WriteAllText($restoreTemp, $text, (New-Object Text.UTF8Encoding($false)))
  Move-Item -LiteralPath $restoreTemp -Destination $restore -Force
  if ($Action -eq 'install') {
    & $binary install --agent $Agent --server $Server --ticket $Ticket
  } else { & $binary clean --agent $Agent }
  if ($LASTEXITCODE -ne 0) { throw 'Quick import did not complete. Review the message above.' }
} finally {
  Remove-Item -LiteralPath $download -ErrorAction SilentlyContinue
  Remove-Item -LiteralPath $restoreTemp -ErrorAction SilentlyContinue
}
