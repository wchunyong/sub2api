import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ModelPlazaContent from '../ModelPlazaContent.vue'
import PlazaGroupSection from '../PlazaGroupSection.vue'
import PlazaFilterBar from '../PlazaFilterBar.vue'
import type { ModelPlazaGroup, ModelPlazaResponse } from '@/api/modelPlaza'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ isAuthenticated: false }) }))

function group(platform: string, rate: number): ModelPlazaGroup {
  return {
    id: platform === 'openai' ? 2 : platform === 'kimi' ? 3 : platform === 'zhipu' ? 4 : 1,
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
    global: { stubs: { PlazaFilterBar: true, PlazaGroupSection: true, Icon: true } }
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

  it('uses neutral pricing copy when the visible domestic groups do not all have a 0.3 rate', () => {
    const wrapper = render([group('deepseek', 0.3), group('kimi', 0.5)])
    expect(wrapper.text()).toContain('modelPlaza.hero.priceLabel')
    expect(wrapper.text()).toContain('modelPlaza.hero.transparent')
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

  it('lists only API groups and models in the cards and filters', async () => {
    const wrapper = render([group('openai', 0.5), group('kimi', 0.3), group('deepseek', 0.3)])
    const filters = wrapper.findComponent(PlazaFilterBar)
    expect(filters.props('platforms')).toEqual(['deepseek', 'kimi', 'openai'])
    expect(filters.props('groups').map((item: { name: string }) => item.name)).toEqual(['openai', 'kimi', 'deepseek'])
    expect(filters.props('rates')).toEqual([0.3, 0.5])
    expect(wrapper.findAllComponents(PlazaGroupSection)).toHaveLength(3)
    expect(wrapper.get('[data-testid="model-count"]').text()).toBe('3')
    filters.vm.$emit('update:platform', 'kimi')
    await wrapper.vm.$nextTick()
    expect(wrapper.findAllComponents(PlazaGroupSection).map(section => section.props('group').platform)).toEqual(['kimi'])
    expect(wrapper.get('[data-testid="model-count"]').text()).toBe('1')
    wrapper.unmount()
  })

  it('removes the live 0.6 Zhipu group and rate', () => {
    const zhipu = group('zhipu', 0.6)
    zhipu.name = '智谱-满血'
    const wrapper = render([zhipu])
    const filters = wrapper.findComponent(PlazaFilterBar)
    expect(filters.props('groups').some((item: { id: number }) => item.id === zhipu.id)).toBe(false)
    expect(filters.props('rates')).not.toContain(0.6)
    expect(wrapper.findComponent(PlazaGroupSection).exists()).toBe(false)
    expect(wrapper.get('[data-testid="model-count"]').text()).toBe('0')
    wrapper.unmount()

    zhipu.user_rate_multiplier = 0.3
    const personalized = render([zhipu])
    expect(personalized.findComponent(PlazaGroupSection).exists()).toBe(false)
    personalized.unmount()
  })

  it('counts distinct visible card names and updates the count after filters and search', async () => {
    const openai = group('openai', 0.5)
    const duplicate = group('openai', 0.3)
    duplicate.id = 5
    const deepseek = group('deepseek', 0.3)
    deepseek.models[0].name = 'shared-model'
    duplicate.models[0].name = 'shared-model'
    const wrapper = render([openai, duplicate, deepseek])
    const filters = wrapper.findComponent(PlazaFilterBar)

    expect(wrapper.get('[data-testid="model-count"]').text()).toBe('2')
    filters.vm.$emit('update:platform', 'deepseek')
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[data-testid="model-count"]').text()).toBe('1')
    filters.vm.$emit('update:search', 'no-match')
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[data-testid="model-count"]').text()).toBe('0')
    wrapper.unmount()
  })

  it('shows no fabricated models, platforms, groups, or rates for an empty API response', () => {
    const wrapper = render([])
    const filters = wrapper.findComponent(PlazaFilterBar)
    expect(filters.props('platforms')).toEqual([])
    expect(filters.props('groups')).toEqual([])
    expect(filters.props('rates')).toEqual([])
    expect(wrapper.get('[data-testid="model-count"]').text()).toBe('0')
    expect(wrapper.findComponent(PlazaGroupSection).exists()).toBe(false)
    expect(wrapper.text()).toContain('modelPlaza.hero.priceLabel')
    wrapper.unmount()
  })
})
