import { describe, expect, it } from 'vitest'
import { isClearlyNonChatModel, orderImportModels, supportsCcs } from '../quickImport'

describe('quick import model screening', () => {
  it('removes dedicated media APIs while retaining chat-capable and unknown models', () => {
    expect(isClearlyNonChatModel('models/gemini-3.1-flash-image')).toBe(true)
    expect(isClearlyNonChatModel('gpt-image-1')).toBe(true)
    expect(isClearlyNonChatModel('gpt-4o-transcribe')).toBe(true)
    expect(isClearlyNonChatModel('deepseek-chat')).toBe(false)
    expect(isClearlyNonChatModel('gpt-audio')).toBe(false)
    expect(isClearlyNonChatModel('custom-model')).toBe(false)
  })

  it('keeps review-only Codex models available without making them the startup default', () => {
    expect(orderImportModels(['codex-auto-review', 'gpt-6.1-sol', 'gpt-6-astra'], true))
      .toEqual(['gpt-6.1-sol', 'gpt-6-astra', 'codex-auto-review'])
    expect(orderImportModels(['codex-auto-review'], true)).toEqual(['codex-auto-review'])
    expect(orderImportModels(['codex-auto-review', 'gpt-6.1-sol'], false))
      .toEqual(['codex-auto-review', 'gpt-6.1-sol'])
  })
})

describe('CC Switch agent support', () => {
  it('limits imports to Claude Code, Codex, OpenCode, and Pi', () => {
    expect(supportsCcs('claude', 'anthropic')).toBe(true)
    expect(supportsCcs('codex', 'openai')).toBe(true)
    expect(supportsCcs('opencode', 'deepseek')).toBe(true)
    expect(supportsCcs('pi', 'deepseek')).toBe(true)
    expect(supportsCcs('workbuddy', 'deepseek')).toBe(false)
    expect(supportsCcs('deepseek-harness', 'deepseek')).toBe(false)
    expect(supportsCcs('grok', 'grok')).toBe(false)
    expect(supportsCcs('gemini', 'gemini')).toBe(false)
  })
})
