import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ModelPlazaContent from '../ModelPlazaContent.vue'
import PlazaGroupSection from '../PlazaGroupSection.vue'
import PlazaFilterBar from '../PlazaFilterBar.vue'
import PlazaDomesticCatalog from '../PlazaDomesticCatalog.vue'
import type { ModelPlazaGroup, ModelPlazaResponse } from '@/api/modelPlaza'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ isAuthenticated: false }) }))

function group(platform: string, rate: number): ModelPlazaGroup {
  return {
    id: platform === 'openai' ? 2 : 1,
    name: platform,
    description: '',
    platform,
    subscription_type: 'standard',
    rate_multiplier: rate,
    peak_rate_enabled: false,
    peak_start: '',
    peak_end: '',
    peak_rate_multiplier: 1,
    is_exclusive: false,
    image_rate_independent: false,
    image_rate_multiplier: 1,
    video_rate_independent: false,
    video_rate_multiplier: 1,
    long_context_pricing_enabled: true,
    models: [{ name: `${platform}-model`, platform, pricing: null, official_pricing: null }]
  }
}

function render(groups: ModelPlazaGroup[], description = '') {
  const response: ModelPlazaResponse = { description, groups }
  return mount(ModelPlazaContent, {
    props: { response, loading: false },
    global: { stubs: { PlazaFilterBar: true, PlazaGroupSection: true, PlazaDomesticCatalog: true, Icon: true } }
  })
}

describe('model plaza promotion headline', () => {
  it('does not render the global preview description', () => {
    const wrapper = render([], '预览说明：内部文案')
    expect(wrapper.text()).not.toContain('预览说明')
    wrapper.unmount()
  })

  it('announces the domestic offer when domestic groups are 0.3 and OpenAI uses its own 0.5 rate', () => {
    const wrapper = render([group('deepseek', 0.3), group('openai', 0.5)])
    expect(wrapper.text()).toContain('modelPlaza.hero.promoLabel')
    expect(wrapper.text()).toContain('modelPlaza.hero.discount')
    wrapper.unmount()
  })

  it('does not claim every domestic model is discounted when a domestic group has another rate', () => {
    const wrapper = render([group('deepseek', 0.3), group('kimi', 0.5)])
    expect(wrapper.text()).toContain('modelPlaza.hero.plannedPromoLabel')
    expect(wrapper.text()).not.toContain('modelPlaza.hero.promoLabel')
    wrapper.unmount()
  })

  it('hides the two superseded DeepSeek cards from the live groups', () => {
    const deepseek = group('deepseek', 0.3)
    deepseek.models = [
      { name: 'deepseek-v4-flash', platform: 'deepseek', pricing: null, official_pricing: null },
      { name: 'deepseek-v4-pro', platform: 'deepseek', pricing: null, official_pricing: null },
      { name: 'deepseek-v4.1-flash', platform: 'deepseek', pricing: null, official_pricing: null }
    ]
    const wrapper = render([deepseek])
    expect(wrapper.findComponent(PlazaGroupSection).props('group').models.map((m: { name: string }) => m.name))
      .toEqual(['deepseek-v4.1-flash'])
    wrapper.unmount()
  })

  it('lets the platform and group filters select Hunyuan planned cards', async () => {
    const wrapper = render([group('openai', 0.5)])
    const filters = wrapper.findComponent(PlazaFilterBar)
    expect(filters.props('platforms')).toContain('hunyuan')
    const hunyuan = filters.props('groups').find((item: { platform: string }) => item.platform === 'hunyuan')
    expect(hunyuan?.rate).toBe(0.3)
    filters.vm.$emit('update:groupId', hunyuan.id)
    await wrapper.vm.$nextTick()
    expect(wrapper.findComponent(PlazaDomesticCatalog).props('families').map((family: { platform: string }) => family.platform))
      .toEqual(['hunyuan'])
    expect(wrapper.findComponent(PlazaGroupSection).exists()).toBe(false)
    wrapper.unmount()
  })

  it('removes the live 0.6 Zhipu group and rate while retaining planned GLM cards', () => {
    const zhipu = group('zhipu', 0.6)
    zhipu.name = '智谱-满血'
    const wrapper = render([zhipu])
    const filters = wrapper.findComponent(PlazaFilterBar)
    expect(filters.props('groups').some((item: { id: number }) => item.id === zhipu.id)).toBe(false)
    expect(filters.props('rates')).not.toContain(0.6)
    expect(wrapper.findComponent(PlazaGroupSection).exists()).toBe(false)
    expect(wrapper.findComponent(PlazaDomesticCatalog).props('families')
      .some((family: { platform: string }) => family.platform === 'zhipu')).toBe(true)
    wrapper.unmount()

    zhipu.user_rate_multiplier = 0.3
    const personalized = render([zhipu])
    expect(personalized.findComponent(PlazaGroupSection).exists()).toBe(false)
    personalized.unmount()
  })

  it('counts distinct visible card names and updates the count after filters and search', async () => {
    const openai = group('openai', 0.5)
    const duplicate = group('openai', 0.3)
    duplicate.id = 3
    const deepseek = group('deepseek', 0.3)
    deepseek.models[0].name = 'deepseek-v4.1-flash'
    const wrapper = render([openai, duplicate, deepseek])
    const filters = wrapper.findComponent(PlazaFilterBar)

    expect(wrapper.get('[data-testid="model-count"]').text()).toBe('13')
    filters.vm.$emit('update:platform', 'hunyuan')
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[data-testid="model-count"]').text()).toBe('2')
    filters.vm.$emit('update:search', 'hy3')
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[data-testid="model-count"]').text()).toBe('1')
    wrapper.unmount()
  })
})
