import type { GroupPlatform } from '@/types'

export type ImportAgent = 'claude' | 'codex' | 'codex-ws' | 'opencode' | 'gemini' | 'grok' | 'deepseek-harness' | 'workbuddy' | 'pi'
export const importAgents: { id: ImportAgent; name: string }[] = [
  { id: 'claude', name: 'Claude Code' }, { id: 'codex', name: 'Codex' },
  { id: 'deepseek-harness', name: 'DeepSeek Harness' }, { id: 'workbuddy', name: 'WorkBuddy' },
  { id: 'opencode', name: 'OpenCode' }, { id: 'pi', name: 'Pi' }
]

// /v1/models reports models available to a key, including dedicated media APIs.
// Only rule out unmistakable non-chat families; aliases and unknown names remain selectable.
export function isClearlyNonChatModel(model: string): boolean {
  const id = model.trim().toLowerCase().split('/').pop() || ''
  return /^(?:gpt-image-|dall-e-|grok-imagine(?:-|$)|imagen-|veo-|sora-|text-embedding-|whisper(?:-|$)|tts-|gpt-transcribe(?:-|$)|gpt-live-transcribe(?:-|$)|gpt-realtime-whisper(?:-|$)|gpt-4o(?:-mini)?-(?:transcribe|tts)(?:-|$)|omni-moderation(?:-|$)|text-moderation(?:-|$))/.test(id) ||
    /^gemini-.*-image(?:-|$)/.test(id) || /^gemini-nano-banana(?:-|$)/.test(id)
}

// The Codex catalog may include a review-only entry before interactive models.
// Keep it available, but do not use it as the startup model when another exists.
export function orderImportModels(models: string[], codex: boolean): string[] {
  if (!codex) return models
  const isReviewOnly = (id: string) => /^codex-auto-review(?:-|$)/.test(id.toLowerCase().split('/').pop() || '')
  return [...models.filter(id => !isReviewOnly(id)), ...models.filter(isReviewOnly)]
}
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
  // The site's CCS flow only exposes clients with a supported provider link.
  // The caller also checks the key's platform and Agent compatibility.
  return !!platform && ['claude', 'codex', 'opencode', 'pi'].includes(agent)
}
