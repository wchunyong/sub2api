import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import QuickImportModal from '../QuickImportModal.vue'

vi.mock('vue-i18n', async (original) => ({ ...await original<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
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
})
