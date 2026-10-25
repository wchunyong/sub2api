import { describe, expect, it } from 'vitest'
import { isClearlyNonChatModel } from '../quickImport'

describe('quick import model screening', () => {
  it('removes dedicated media APIs while retaining chat-capable and unknown models', () => {
    expect(isClearlyNonChatModel('models/gemini-3.1-flash-image')).toBe(true)
    expect(isClearlyNonChatModel('gpt-image-1')).toBe(true)
    expect(isClearlyNonChatModel('gpt-4o-transcribe')).toBe(true)
    expect(isClearlyNonChatModel('deepseek-chat')).toBe(false)
    expect(isClearlyNonChatModel('gpt-audio')).toBe(false)
    expect(isClearlyNonChatModel('custom-model')).toBe(false)
  })
})
