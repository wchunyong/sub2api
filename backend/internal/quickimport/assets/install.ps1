param(
  [Parameter(Mandatory=$true)][ValidateSet('install','clean')][string]$Action,
  [Parameter(Mandatory=$true)][ValidateSet('claude','codex','opencode')][string]$Agent,
  [string]$Server,
  [string]$Ticket
)
$ErrorActionPreference = 'Stop'
$runtime = Get-Command python -ErrorAction SilentlyContinue
if (-not $runtime) { throw 'Install Python 3.11 or later from python.org, reopen PowerShell and retry.' }
& $runtime.Source -c 'import sys; sys.exit(0 if sys.version_info >= (3,11) else 1)'
if ($LASTEXITCODE -ne 0) { throw 'Python 3.11 or later is required.' }
if ($Action -eq 'clean') {
  $runner = Join-Path $env:USERPROFILE ".sub2api-quick-import/$Agent/restore.py"
  if (-not (Test-Path -LiteralPath $runner)) { throw 'No local recovery script for this Agent.' }
  & $runtime.Source $runner clean --agent $Agent
} else {
  $uri = [Uri]$Server
  if ($uri.Scheme -ne 'https' -or $uri.UserInfo -or $uri.Query -or $uri.Fragment) { throw 'HTTPS server required.' }
  $runner = Join-Path ([IO.Path]::GetTempPath()) ([IO.Path]::GetRandomFileName() + '.py')
  try {
    Invoke-WebRequest -UseBasicParsing -MaximumRedirection 0 -Uri ($Server.TrimEnd('/') + '/api/v1/quick-import/assets/installer.py') -OutFile $runner
    & $runtime.Source $runner install --agent $Agent --server $Server --ticket $Ticket
  } finally { Remove-Item -LiteralPath $runner -ErrorAction SilentlyContinue }
}
if ($LASTEXITCODE -ne 0) { throw 'Quick import did not complete. Review the message above.' }
