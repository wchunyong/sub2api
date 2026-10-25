import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import UseKeyModal from '../UseKeyModal.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copyToClipboard: vi.fn().mockResolvedValue(true) }) }))
vi.mock('file-saver', () => ({ saveAs: vi.fn() }))

function render(client: string, platform: 'openai' | 'deepseek' = 'deepseek') {
  return mount(UseKeyModal, {
    props: { show: true, apiKey: 'sk-example', baseUrl: 'https://example.com', platform, client, embedded: true },
    global: { stubs: { Icon: { template: '<span />' } } }
  })
}

function blocks(wrapper: ReturnType<typeof render>) {
  return wrapper.findAll('pre code').map(block => block.text())
}

describe('manual import all models', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('defaults Codex to all manifest models and embeds the key in config.toml', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ models: [{ slug: 'gpt-first' }, { slug: 'gpt-second' }] }) }))
    const wrapper = render('codex', 'openai')
    await flushPromises()
    expect(wrapper.get('[data-testid="manual-model-list"]').findAll('input:checked')).toHaveLength(2)
    expect(wrapper.find('[data-testid="manual-model-required"]').exists()).toBe(false)
    const config = blocks(wrapper).find(block => block.includes('model_provider = "OpenAI"')) || ''
    expect(config).toContain('model = "gpt-first"')
    expect(config).toContain('experimental_bearer_token = "sk-example"')
    expect(config).not.toContain('env_key =')
    expect(wrapper.text()).not.toContain('Terminal')
    expect(wrapper.text()).not.toContain('PowerShell')
    wrapper.unmount()
  })

  it('writes only selected Codex models into a local model catalog', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ models: [{ slug: 'gpt-first' }, { slug: 'gpt-second' }] }) }))
    const wrapper = render('codex', 'openai')
    await flushPromises()
    await wrapper.get('[data-testid="manual-model-list"] input[value="gpt-first"]').setValue(false)
    const config = blocks(wrapper).find(block => block.includes('model_provider = "OpenAI"')) || ''
    expect(config).toContain('model = "gpt-second"')
    expect(config).toContain('model_catalog_json = "~/.codex/codex-models.json"')
    expect(config).not.toContain('model_catalog_url =')
    const catalog = JSON.parse(blocks(wrapper).find(block => block.includes('"models"')) || '{}')
    expect(catalog.models.map((model: { slug: string }) => model.slug)).toEqual(['gpt-second'])
    await wrapper.setProps({ apiKey: 'sk-new-key' })
    await flushPromises()
    const resetConfig = blocks(wrapper).find(block => block.includes('model_provider = "OpenAI"')) || ''
    expect(resetConfig).toContain('model_catalog_url = "https://example.com/v1/models"')
    expect(resetConfig).not.toContain('model_catalog_json =')
    wrapper.unmount()
  })

  it('includes every chosen OpenCode model and supports deselection', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: [{ id: 'first-chat' }, { id: 'second-chat' }] }) }))
    const wrapper = render('opencode')
    await flushPromises()
    let parsed = JSON.parse(blocks(wrapper)[0]!)
    expect(Object.keys(parsed.provider.lianjieai.models)).toEqual(['first-chat', 'second-chat'])
    await wrapper.get('[data-testid="manual-model-list"] input[value="first-chat"]').setValue(false)
    parsed = JSON.parse(blocks(wrapper)[0]!)
    expect(Object.keys(parsed.provider.lianjieai.models)).toEqual(['second-chat'])
    wrapper.unmount()
  })

  it.each([
    ['deepseek-harness', 'cordis.patch.yml'],
    ['workbuddy', 'models.json'],
    ['pi', 'models.json']
  ])('generates an in-page %s file with all group models', async (client, filename) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: [{ id: 'deepseek-chat' }, { id: 'deepseek-reasoner' }] }) }))
    const wrapper = render(client)
    await flushPromises()
    expect(wrapper.get('[data-testid="manual-model-list"]').findAll('input:checked')).toHaveLength(2)
    expect(wrapper.text()).toContain(filename)
    const config = blocks(wrapper).join('\n')
    expect(config).toContain('deepseek-chat')
    expect(config).toContain('deepseek-reasoner')
    expect(config).toContain('sk-example')
    expect(wrapper.text()).not.toContain('Terminal')
    if (client === 'workbuddy') expect(JSON.parse(config)).toHaveLength(2)
    if (client === 'pi') expect(JSON.parse(config).providers.lianjieai.models).toHaveLength(2)
    await wrapper.findAll('button').find(button => button.text().trim() === 'Windows')!.trigger('click')
    expect(wrapper.text()).toContain('%USERPROFILE%')
    wrapper.unmount()
  })
})
