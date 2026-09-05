import { describe, it, expect, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import QuickImportModal from '../QuickImportModal.vue'

vi.mock('vue-i18n', async (original) => ({ ...await original<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copyToClipboard: vi.fn().mockResolvedValue(true) }) }))
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
  it('selects an Agent before showing actions and carries it into manual configuration', async () => {
    const wrapper = render()
    expect(wrapper.find('[data-testid="manual"]').exists()).toBe(false)
    await wrapper.get('[data-testid="agent-opencode"]').trigger('click')
    await wrapper.get('[data-testid="manual"]').trigger('click')
    expect(wrapper.get('[data-testid="manual-client"]').text()).toBe('opencode')
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
    await wrapper.get('[data-testid="auto"]').trigger('click')
    await wrapper.get('[data-testid="generate"]').trigger('click')
    await wrapper.setProps({ keyId: 2 })
    resolve({ ticket: 'a'.repeat(64), agent: 'opencode', model: 'mock', expires_in: 300 })
    await flushPromises()
    expect(wrapper.find('pre').exists()).toBe(false)
    wrapper.unmount()
  })
  it('allows offline cleanup for an inactive key and only selects its Agent', async () => {
    const wrapper = render({ active: false })
    await wrapper.get('[data-testid="agent-claude"]').trigger('click')
    await wrapper.get('[data-testid="clean"]').trigger('click')
    expect(wrapper.get('pre').text()).toContain('/claude/restore.py')
    expect(wrapper.get('pre').text()).not.toContain('/opencode/')
    wrapper.unmount()
  })
})
