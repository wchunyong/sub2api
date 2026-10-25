import { beforeEach, describe, it, expect, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import QuickImportModal from '../QuickImportModal.vue'
import { CC_SWITCH_USAGE_SCRIPT } from '@/utils/ccswitchImport'

vi.mock('vue-i18n', async (original) => ({ ...await original<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
const { copy } = vi.hoisted(() => ({ copy: vi.fn().mockResolvedValue(true) }))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copyToClipboard: copy }) }))
const { createTicket } = vi.hoisted(() => ({ createTicket: vi.fn() }))
vi.mock('@/api/quickImport', () => ({ createImportTicket: createTicket, importServer: () => 'https://example.com' }))
const props = { show: true, apiKey: 'sk-test', keyId: 1, baseUrl: 'https://example.com', platform: 'openai' as const }
function render(extra = {}) {
  return mount(QuickImportModal, { props: { ...props, ...extra }, global: { stubs: {
    BaseDialog: { template: '<div><slot /></div>' },
    UseKeyModal: { props: ['client'], template: '<div data-testid="manual-client">{{ client }}</div>' }
  } } })
}
describe('QuickImportModal', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation(async (url: string) => ({
      ok: true,
      json: async () => url.includes('client_version=')
        ? { models: [{ slug: 'gpt-5.5' }] }
        : { data: [{ id: 'gpt-5.5' }] }
    })))
  })
  it.each([
    ['claude', 'deepseek'], ['codex', 'openai'], ['opencode', 'deepseek']
  ])('enables CCS for %s with a compatible key', async (agent, platform) => {
    const wrapper = render({ platform })
    await wrapper.get(`[data-testid="agent-${agent}"]`).trigger('click')
    await flushPromises()
    expect(wrapper.find('#quick-import-model').exists()).toBe(false)
    if (agent === 'opencode') {
      expect(wrapper.get('[data-testid="ccs"]').attributes('disabled')).toBeDefined()
      await wrapper.get('[data-testid="ccs-action-model-select"]').setValue('gpt-5.5')
    }
    expect(wrapper.get('[data-testid="ccs"]').attributes('disabled')).toBeUndefined()
    expect(wrapper.findAll('.qi-choice')).toHaveLength(3)
    const assign = vi.fn()
    vi.stubGlobal('window', Object.create(window, { location: { value: { assign } } }))
    try {
      await wrapper.get('[data-testid="ccs"]').trigger('click')
      expect(assign).toHaveBeenCalledOnce()
      const params = new URL(assign.mock.calls[0][0]).searchParams
      expect(params.get('app')).toBe(agent)
      expect(params.get('endpoint')).toBe(agent === 'claude' ? props.baseUrl : `${props.baseUrl}/v1`)
      expect(params.get('model')).toBe('gpt-5.5')
      expect(params.get('apiFormat')).toBe(agent === 'claude' ? 'anthropic' : agent === 'codex' ? 'openai_responses' : 'openai_chat')
      expect(atob(params.get('usageScript') || '')).toBe(CC_SWITCH_USAGE_SCRIPT)
    } finally {
      vi.unstubAllGlobals()
    }
    wrapper.unmount()
  })
  it('keeps CCS disabled for an inactive key', async () => {
    const wrapper = render({ active: false, platform: 'openai' })
    await wrapper.get('[data-testid="agent-codex"]').trigger('click')
    expect(wrapper.get('[data-testid="ccs"]').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
  it('opens CCS directly from the Claude and Codex cards without a second step', async () => {
    const assign = vi.fn()
    vi.stubGlobal('window', Object.create(window, { location: { value: { assign, origin: 'https://example.com' } } }))
    try {
      for (const [agent, platform] of [['claude', 'anthropic'], ['codex', 'openai']] as const) {
        const wrapper = render({ ccsOnly: true, platform })
        await flushPromises()
        expect(wrapper.find('[data-testid="agent-workbuddy"]').exists()).toBe(false)
        expect(wrapper.find('[data-testid="agent-deepseek-harness"]').exists()).toBe(false)
        await wrapper.get(`[data-testid="agent-${agent}"]`).trigger('click')
        expect(wrapper.find('[data-testid="auto"]').exists()).toBe(false)
        expect(wrapper.find('[data-testid="manual"]').exists()).toBe(false)
        const params = new URL(assign.mock.lastCall?.[0]).searchParams
        expect(params.get('app')).toBe(agent)
        expect(params.get('model')).toBe('gpt-5.5')
        expect(params.get('config')).not.toBeNull()
        wrapper.unmount()
      }
    } finally { vi.unstubAllGlobals() }
  })
  it('selects one OpenCode or Pi model inline before CCS import', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: [{ id: 'first-chat' }, { id: 'second-chat' }] }) }))
    const assign = vi.fn()
    vi.stubGlobal('window', Object.create(window, { location: { value: { assign, origin: 'https://example.com' } } }))
    try {
      for (const agent of ['opencode', 'pi'] as const) {
        const wrapper = render({ ccsOnly: true, platform: 'deepseek' })
        await flushPromises()
        await wrapper.get(`[data-testid="agent-${agent}"]`).trigger('click')
        expect(wrapper.find('[data-testid="auto"]').exists()).toBe(false)
        expect(assign).not.toHaveBeenCalled()
        await wrapper.get('[data-testid="ccs-model-select"]').setValue('second-chat')
        await wrapper.get('[data-testid="ccs-inline-submit"]').trigger('click')
        const params = new URL(assign.mock.lastCall?.[0]).searchParams
        expect(params.get('app')).toBe(agent)
        expect(params.get('model')).toBe('second-chat')
        wrapper.unmount()
        assign.mockClear()
      }
    } finally { vi.unstubAllGlobals() }
  })
  it('only offers Codex manifest models, not generic-only models', async () => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation(async (url: string) => ({
      ok: true,
      json: async () => url.includes('client_version=')
        ? { models: [{ slug: 'gpt-codex-supported' }] }
        : { data: [{ id: 'generic-only' }] }
    })))
    const wrapper = render()
    await wrapper.get('[data-testid="agent-codex"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('#quick-import-model').exists()).toBe(false)
    expect(wrapper.get('[data-testid="auto"]').attributes('disabled')).toBeUndefined()
    createTicket.mockResolvedValueOnce({ ticket: 'c'.repeat(64), agent: 'codex', expires_in: 300 })
    await wrapper.get('[data-testid="auto"]').trigger('click')
    expect(createTicket).toHaveBeenCalledWith(1, 'codex', 'gpt-codex-supported', expect.any(AbortSignal))
    wrapper.unmount()
  })
  it('uses the first chat model as the default without requiring a selection', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: [{ id: 'gpt-image-1' }, { id: 'deepseek-chat' }] }) }))
    const wrapper = render({ platform: 'deepseek' })
    await wrapper.get('[data-testid="agent-workbuddy"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('#quick-import-model').exists()).toBe(false)
    expect(wrapper.get('[data-testid="auto"]').attributes('disabled')).toBeUndefined()
    createTicket.mockResolvedValueOnce({ ticket: 'd'.repeat(64), agent: 'workbuddy', expires_in: 300 })
    await wrapper.get('[data-testid="auto"]').trigger('click')
    expect(createTicket).toHaveBeenCalledWith(1, 'workbuddy', 'deepseek-chat', expect.any(AbortSignal))
    wrapper.unmount()
  })
  it('does not issue a ticket when model discovery fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }))
    const wrapper = render({ platform: 'deepseek' })
    await wrapper.get('[data-testid="agent-workbuddy"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('#quick-import-model').exists()).toBe(false)
    expect(wrapper.get('[data-testid="auto"]').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('keys.quickImport.modelUnavailable')
    wrapper.unmount()
  })
  it('explains when a key has no chat models', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: [{ id: 'gpt-image-1' }] }) }))
    const wrapper = render({ platform: 'openai' })
    await wrapper.get('[data-testid="agent-workbuddy"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('keys.quickImport.modelNoChatAvailable')
    expect(wrapper.get('[data-testid="auto"]').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
  it('does not offer Codex for groups without a Codex model catalog', () => {
    const wrapper = render({ platform: 'deepseek' })
    expect(wrapper.get('[data-testid="agent-codex"]').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
  it('clears the copy error on a successful retry', async () => {
    createTicket.mockResolvedValueOnce({ ticket: 'b'.repeat(64), agent: 'codex', expires_in: 300 })
    copy.mockResolvedValueOnce(false).mockResolvedValueOnce(true)
    const wrapper = render()
    await wrapper.get('[data-testid="agent-codex"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="auto"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    const retry = wrapper.findAll('button').find(button => button.text() === 'keys.quickImport.copyImport')!
    await retry.trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    wrapper.unmount()
  })
  it('disables unsupported agents and copies directly without settings', async () => {
    const wrapper = render()
    expect(wrapper.find('[data-testid="agent-gemini"]').exists()).toBe(false)
    expect(wrapper.findAll('.qi-agent')).toHaveLength(6)
    expect(wrapper.text()).toContain('Codex')
    createTicket.mockResolvedValueOnce({ ticket: 'a'.repeat(64), agent: 'codex', expires_in: 300 })
    await wrapper.get('[data-testid="agent-codex"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="auto"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('select').exists()).toBe(false)
    expect(wrapper.find('input').exists()).toBe(false)
    expect(createTicket).toHaveBeenCalledWith(1, 'codex', 'gpt-5.5', expect.any(AbortSignal))
    expect(copy.mock.lastCall?.[0]).toContain('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa')
    wrapper.unmount()
  })
  it('selects an Agent before showing actions and carries it into manual configuration', async () => {
    const wrapper = render()
    expect(wrapper.find('[data-testid="manual"]').exists()).toBe(false)
    await wrapper.get('[data-testid="agent-opencode"]').trigger('click')
    await wrapper.get('[data-testid="manual"]').trigger('click')
    expect(wrapper.get('[data-testid="manual-client"]').text()).toBe('opencode')
  })
  it('opens the configured documentation for agents with a separate manual guide', async () => {
    const open = vi.spyOn(window, 'open').mockImplementation(() => null)
    try {
      const wrapper = render({ docUrl: 'https://guide.example.com/zh/' })
      await wrapper.get('[data-testid="agent-workbuddy"]').trigger('click')
      await wrapper.get('[data-testid="manual"]').trigger('click')
      expect(open).toHaveBeenCalledWith('https://guide.example.com/zh/agent/workbuddy', '_blank', 'noopener,noreferrer')
      wrapper.unmount()
    } finally {
      open.mockRestore()
    }
  })
  it('honors CCS hiding and resets the selection when the key changes', async () => {
    const wrapper = render({ hideCcs: true })
    await wrapper.get('[data-testid="agent-codex"]').trigger('click')
    expect(wrapper.find('[data-testid="ccs"]').exists()).toBe(false)
    await wrapper.setProps({ keyId: 2, apiKey: 'sk-another' })
    expect(wrapper.find('[data-testid="manual"]').exists()).toBe(false)
  })
  it('discards an in-flight command when the Agent changes', async () => {
    let resolve!: (value: unknown) => void
    createTicket.mockImplementationOnce(() => new Promise(r => { resolve = r }))
    const wrapper = render()
    await wrapper.get('[data-testid="agent-opencode"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="auto"]').trigger('click')
    await wrapper.setProps({ keyId: 2 })
    resolve({ ticket: 'a'.repeat(64), agent: 'opencode', model: 'mock', expires_in: 300 })
    await flushPromises()
    expect(wrapper.find('pre').exists()).toBe(false)
    wrapper.unmount()
  })
  it('allows offline cleanup for an inactive key and only selects its Agent', async () => {
    const wrapper = render({ active: false, platform: 'anthropic' })
    await wrapper.get('[data-testid="agent-claude"]').trigger('click')
    await wrapper.get('[data-testid="clean"]').trigger('click')
    await flushPromises()
    expect(copy.mock.lastCall?.[0]).toContain('/claude/restore.sh')
    wrapper.unmount()
  })
})
