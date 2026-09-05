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
    return `& ([scriptblock]::Create((irm -MaximumRedirection 0 ${ps(root + '/setup/' + agent + '.ps1')}))) ${ps(ticket)}`
  }
  return `curl -fsS --proto '=https' ${sh(root + '/setup/' + agent + '.sh')} | sh -s -- ${sh(ticket)}`
}

export function cleanupCommand(os: ImportOS, agent: ImportAgent, server?: string): string {
  checkedAgent(agent)
  const local = os === 'windows'
    ? `powershell -NoProfile -ExecutionPolicy Bypass -File "$env:USERPROFILE/.sub2api-quick-import/${agent}/restore.ps1"`
    : `sh "$HOME/.sub2api-quick-import/${agent}/restore.sh"`
  if (!server) return local
  const url = new URL(server)
  if (url.protocol !== 'https:' || url.username || url.password || url.search || url.hash) throw new Error('Invalid server')
  const root = server.replace(/\/+$/, '')
  return os === 'windows'
    ? `& { if (Test-Path "$env:USERPROFILE/.sub2api-quick-import/${agent}/restore.ps1") { ${local}; if ($LASTEXITCODE -eq 0) { return } }; irm -MaximumRedirection 0 ${ps(root + '/setup/' + agent + '-clean.ps1')} | iex }`
    : `if [ -f "$HOME/.sub2api-quick-import/${agent}/restore.sh" ] && ${local}; then :; else curl -fsS --proto '=https' ${sh(root + '/setup/' + agent + '-clean.sh')} | sh; fi`
}
