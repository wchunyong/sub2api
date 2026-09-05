$ErrorActionPreference = 'Stop'
$sourceDir = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$testAssets = Join-Path $sourceDir 'native-assets'
$testNative = Join-Path $testAssets 'quick-import-windows-amd64.exe'
$testRoot = Join-Path ([IO.Path]::GetTempPath()) ('lianjieai-native-' + [Guid]::NewGuid().ToString('N'))
$priorHome = $env:USERPROFILE
$priorPath = $env:PATH
New-Item -ItemType Directory -Path $testRoot | Out-Null
try {
  $env:USERPROFILE = $testRoot
  $env:PATH = "$env:SystemRoot\System32;$env:SystemRoot\System32\WindowsPowerShell\v1.0"
  if (Get-Command python -ErrorAction SilentlyContinue) { throw 'Python unexpectedly available in test PATH' }
  foreach ($testAgent in @('claude','codex','opencode')) {
    $payload = @{version=1;agent=$testAgent;api_key='mock-native-key';base_url='https://example.com/v1';probe_url='https://example.com/v1/models';model='mock-model';protocol='openai'} | ConvertTo-Json -Compress
    $payload | & $testNative install --agent $testAgent --stdin --skip-client-check --home $testRoot
    if ($LASTEXITCODE -ne 0) { throw 'Native install failed' }
    & $testNative clean --agent $testAgent --home $testRoot --yes
    if ($LASTEXITCODE -ne 0) { throw 'Native cleanup failed' }
  }
  function Invoke-WebRequest {
    param($Uri,$OutFile,$Headers,$MaximumRedirection,$TimeoutSec,[switch]$UseBasicParsing)
    $asset = Join-Path $testAssets ([Uri]$Uri).Segments[-1]
    if ($OutFile) { Copy-Item -LiteralPath $asset -Destination $OutFile }
    else { [pscustomobject]@{Content=[IO.File]::ReadAllText($asset)} }
  }
  $payload = @{version=1;agent='opencode';api_key='mock-native-key';base_url='https://example.com/v1';probe_url='https://example.com/v1/models';model='mock-model';protocol='openai'} | ConvertTo-Json -Compress
  $payload | & $testNative install --agent opencode --stdin --skip-client-check --home $testRoot
  if ($LASTEXITCODE -ne 0) { throw 'Native fixture install failed' }
  & (Join-Path $sourceDir 'assets/install.ps1') -Action clean -Agent opencode -Server https://example.com
  if (Test-Path (Join-Path $testRoot '.config/opencode/opencode.json')) { throw 'Launcher cleanup failed' }
  $payload | & $testNative install --agent opencode --stdin --skip-client-check --home $testRoot
  'y' | & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $testRoot '.sub2api-quick-import/opencode/restore.ps1')
  if ($LASTEXITCODE -ne 0 -or (Test-Path (Join-Path $testRoot '.config/opencode/opencode.json'))) { throw 'Offline cleanup failed' }
  Write-Output 'PASS: no Python in PATH; native three-Agent round trips and downloader/offline cleanup'
} finally {
  $env:USERPROFILE = $priorHome
  $env:PATH = $priorPath
  $resolved = [IO.Path]::GetFullPath($testRoot)
  $allowed = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
  if (!$resolved.StartsWith($allowed, [StringComparison]::OrdinalIgnoreCase) -or !(Split-Path $resolved -Leaf).StartsWith('lianjieai-native-')) { throw 'Refusing unsafe test cleanup' }
  Remove-Item -LiteralPath $resolved -Recurse -Force
}
