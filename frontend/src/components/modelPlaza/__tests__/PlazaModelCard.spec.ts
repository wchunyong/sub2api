import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PlazaModelCard from '../PlazaModelCard.vue'
import type { PlazaModel } from '@/api/modelPlaza'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string, params?: { price?: string }) =>
    key === 'modelPlaza.card.originalPrice' ? `原价 ${params?.price}` : key }) }
})

const model: PlazaModel = {
  name: 'deepseek-chat',
  platform: 'deepseek',
  pricing: {
    billing_mode: 'token', input_price: 0.3e-6, output_price: 1.2e-6,
    cache_write_price: null, cache_read_price: 0.03e-6,
    image_input_price: null, image_output_price: null, per_request_price: null, intervals: []
  },
  official_pricing: {
    input_price: 0.5e-6, output_price: 1.2e-6,
    cache_write_price: null, cache_read_price: 0.03e-6
  }
}

describe('PlazaModelCard', () => {
  it('shows channel base and billed token prices using the effective group rate', () => {
    const wrapper = mount(PlazaModelCard, {
      props: { model, rateMultiplier: 0.3 }
    })
    expect(wrapper.text()).toContain('deepseek-chat')
    expect(wrapper.text()).toContain('原价 $0.30')
    expect(wrapper.text()).toContain('$0.09')
    expect(wrapper.text()).not.toContain('$0.50')
    expect(wrapper.text()).toContain('$1.20')
    expect(wrapper.text()).toContain('$0.36')
    expect(wrapper.text()).toContain('$0.009')
    expect(wrapper.text()).toContain('3')
  })

  it('uses a user rate instead of the default group rate', () => {
    const wrapper = mount(PlazaModelCard, {
      props: { model, rateMultiplier: 0.3, userRateMultiplier: 0.2 }
    })
    expect(wrapper.text()).toContain('$0.06')
    expect(wrapper.text()).toContain('$0.24')
    expect(wrapper.text()).toContain('2')
  })

  it('labels a planned 0.3 price separately from billed prices', () => {
    const wrapper = mount(PlazaModelCard, {
      props: { model, rateMultiplier: 0.3, planned: true }
    })
    expect(wrapper.text()).toContain('原价 $0.30')
    expect(wrapper.text()).toContain('modelPlaza.card.plannedPrice')
    expect(wrapper.find('[data-price="input"]').text()).toBe('$0.09')
  })

  it('uses the first configured tier when a per-request model has no flat price', () => {
    const tiered: PlazaModel = {
      ...model,
      name: 'image-model',
      pricing: {
        ...model.pricing!,
        billing_mode: 'image',
        input_price: null,
        output_price: null,
        cache_read_price: null,
        intervals: [{ min_tokens: 0, max_tokens: null, tier_label: 'standard',
          input_price: null, output_price: null, cache_write_price: null,
          cache_read_price: null, per_request_price: 2 }]
      },
      official_pricing: null
    }
    const wrapper = mount(PlazaModelCard, {
      props: { model: tiered, rateMultiplier: 0.3, imageRateIndependent: true, imageRateMultiplier: 0.5 }
    })
    expect(wrapper.find('[data-price="request"]').text()).toBe('$1.00')
    expect(wrapper.text()).toContain('原价 $2.00')
  })
})
