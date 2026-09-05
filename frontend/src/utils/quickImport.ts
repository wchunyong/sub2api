import type { GroupPlatform } from '@/types'

export type ImportAgent = 'claude' | 'codex' | 'codex-ws' | 'opencode' | 'gemini' | 'grok'
export const importAgents: { id: ImportAgent; name: string }[] = [
  { id: 'claude', name: 'Claude Code' }, { id: 'codex', name: 'ChatGPT/Codex' },
  { id: 'opencode', name: 'OpenCode' }, { id: 'gemini', name: 'Gemini CLI' },
  { id: 'grok', name: 'Grok CLI' }, { id: 'codex-ws', name: 'Codex WebSocket' }
]
export function supportsAgent(agent: ImportAgent, platform: GroupPlatform | null, messages = false): boolean {
  if (!platform) return false
  if (agent === 'codex-ws') return platform === 'openai'
  if (agent === 'grok') return platform === 'grok'
  if (agent === 'gemini') return platform === 'gemini' || platform === 'antigravity'
  if (agent === 'claude') return platform !== 'gemini' && (platform !== 'openai' || messages)
  return true
}

export function supportsCcs(agent: ImportAgent, platform: GroupPlatform | null): boolean {
  // CCS supports these clients independently of the key's upstream platform.
  // The caller still checks key validity and Agent compatibility.
  return (['claude', 'codex', 'opencode'].includes(agent) && !!platform) ||
    (agent === 'gemini' && (platform === 'gemini' || platform === 'antigravity')) ||
    (agent === 'grok' && platform === 'grok')
}
