import type { GroupPlatform } from '@/types'

export type ImportAgent = 'claude' | 'codex' | 'codex-ws' | 'opencode' | 'gemini' | 'grok' | 'deepseek-harness' | 'workbuddy' | 'pi'
export const importAgents: { id: ImportAgent; name: string }[] = [
  { id: 'claude', name: 'Claude Code' }, { id: 'codex', name: 'Codex' },
  { id: 'deepseek-harness', name: 'DeepSeek Harness' }, { id: 'workbuddy', name: 'WorkBuddy' },
  { id: 'opencode', name: 'OpenCode' }, { id: 'pi', name: 'Pi' }
]
export function supportsAgent(agent: ImportAgent, platform: GroupPlatform | null, messages = false): boolean {
  if (!platform || !['openai', 'anthropic', 'antigravity', 'gemini', 'deepseek', 'grok', 'kimi', 'zhipu', 'composite'].includes(platform)) return false
  if (agent === 'codex-ws') return platform === 'openai'
  if (agent === 'codex') return platform === 'openai' || platform === 'composite'
  if (agent === 'grok') return platform === 'grok'
  if (agent === 'gemini') return platform === 'gemini' || platform === 'antigravity'
  if (agent === 'claude') return platform !== 'gemini' && (platform !== 'openai' || messages)
  if (['deepseek-harness', 'workbuddy', 'pi'].includes(agent)) return true
  return true
}

export function supportsCcs(agent: ImportAgent, platform: GroupPlatform | null): boolean {
  // CCS supports these clients independently of the key's upstream platform.
  // The caller still checks key validity and Agent compatibility.
  return (['claude', 'codex', 'opencode'].includes(agent) && !!platform) ||
    (agent === 'gemini' && (platform === 'gemini' || platform === 'antigravity')) ||
    (agent === 'grok' && platform === 'grok')
}
