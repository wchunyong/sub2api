import type { ImportAgent } from './quickImport'
export type ImportOS = 'windows' | 'unix'
export function automaticAgent(agent: ImportAgent): boolean { return ['claude', 'codex', 'opencode'].includes(agent) }
function checkedAgent(agent: ImportAgent) { if (!automaticAgent(agent)) throw new Error('Unsupported Agent'); return agent }
const ps = (value: string) => `'${value.replace(/'/g, "''")}'`
const sh = (value: string) => `'${value.replace(/'/g, `'"'"'`)}'`
export function importCommand(os: ImportOS, agent: ImportAgent, server: string, ticket: string): string {
  checkedAgent(agent)
  const url = new URL(server)
  if (url.protocol !== 'https:' || url.username || url.password || url.search || url.hash || !/^[a-f0-9]{64}$/.test(ticket)) throw new Error('Invalid command parameters')
  const root = server.replace(/\/+$/, '')
  if (os === 'windows') {
    return `& { $p = Join-Path $env:TEMP ([IO.Path]::GetRandomFileName() + '.ps1'); try { Invoke-WebRequest -UseBasicParsing -MaximumRedirection 0 -Uri ${ps(root + '/api/v1/quick-import/assets/install.ps1')} -OutFile $p; powershell -NoProfile -ExecutionPolicy Bypass -File $p -Action install -Agent ${ps(agent)} -Server ${ps(root)} -Ticket ${ps(ticket)} } finally { Remove-Item -LiteralPath $p -ErrorAction SilentlyContinue } }`
  }
  return `(p=$(mktemp) && trap 'rm -f "$p"' EXIT HUP INT TERM && curl --fail --silent --show-error --proto '=https' --max-time 30 ${sh(root + '/api/v1/quick-import/assets/install.sh')} -o "$p" && sh "$p" install ${sh(agent)} ${sh(root)} ${sh(ticket)})`
}
export function cleanupCommand(os: ImportOS, agent: ImportAgent): string {
  checkedAgent(agent)
  return os === 'windows'
    ? `python "$env:USERPROFILE/.sub2api-quick-import/${agent}/restore.py" clean --agent ${agent}`
    : `python3 "$HOME/.sub2api-quick-import/${agent}/restore.py" clean --agent ${agent}`
}
