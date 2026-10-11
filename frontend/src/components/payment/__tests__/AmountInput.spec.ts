import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import AmountInput from '../AmountInput.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params && 'amount' in params
        ? `${key} ${String(params.amount)}`
        : key === 'payment.quickAmounts' ? '充值金额' : key,
  }),
}))

describe('AmountInput recharge presets', () => {
  it('shows fixed recharge amount presets without a custom amount input', async () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: null,
        amounts: [10, 20, 50, 100, 200, 400, 800],
      },
    })

    expect(wrapper.text()).toContain('充值金额')
    expect(wrapper.text()).not.toContain('自定义金额')
    expect(wrapper.find('input').exists()).toBe(false)

    const buttons = wrapper.findAll('button')
    expect(buttons.map(button => button.text())).toEqual(['10', '20', '50', '100', '200', '400', '800'])

    await buttons[3].trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([100])
  })
})

describe('recharge bonus hints on quick amounts', () => {
  const tiers = [
    { min_amount: 100, bonus_percent: 20 },
    { min_amount: 500, bonus_percent: 30 },
  ]

  it('renders no badge or second line when no tiers are configured', () => {
    const wrapper = mount(AmountInput, { props: { modelValue: null, amounts: [50, 100, 500] } })
    expect(wrapper.find('[data-testid="quick-amount-bonus-badge"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="quick-amount-credited"]').exists()).toBe(false)
  })

  it('bonus mode: price tag only on amounts that hit a tier, credited totals on every button', () => {
    const wrapper = mount(AmountInput, { props: { modelValue: null, amounts: [50, 100, 500], bonusTiers: tiers } })
    const below = wrapper.get('[data-testid="quick-amount-50"]')
    const first = wrapper.get('[data-testid="quick-amount-100"]')
    const second = wrapper.get('[data-testid="quick-amount-500"]')

    expect(below.find('[data-testid="quick-amount-bonus-badge"]').exists()).toBe(false)
    expect(below.get('[data-testid="quick-amount-credited"]').text()).toContain('$50.00')
    expect(first.get('[data-testid="quick-amount-bonus-badge"]').text()).toBe('+20%')
    expect(first.get('[data-testid="quick-amount-credited"]').text()).toContain('$120.00')
    expect(second.get('[data-testid="quick-amount-bonus-badge"]').text()).toBe('+30%')
    expect(second.get('[data-testid="quick-amount-credited"]').text()).toContain('$650.00')
  })

  it('bonus mode: matches tiers by the entered amount but credits by the multiplier', () => {
    const wrapper = mount(AmountInput, {
      props: { modelValue: null, amounts: [1000], bonusTiers: tiers, multiplier: 0.14 },
    })
    const button = wrapper.get('[data-testid="quick-amount-1000"]')
    expect(button.get('[data-testid="quick-amount-bonus-badge"]').text()).toBe('+30%')
    // 1000 × 0.14 = 140 base, +30% = 182
    expect(button.get('[data-testid="quick-amount-credited"]').text()).toContain('$182.00')
  })

  it('uses fixed recharge credit when bonus tiers apply to a preset', () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: null,
        amounts: [200],
        bonusTiers: tiers,
        creditForAmount: (amount: number) => amount === 200 ? 210 : amount,
      },
    })
    expect(wrapper.get('[data-testid="quick-amount-200"] [data-testid="quick-amount-credited"]').text()).toContain('$252.00')
  })

  it('discount mode: tag reads N% OFF and the second line shows the discounted payment', () => {
    const wrapper = mount(AmountInput, {
      props: { modelValue: null, amounts: [50, 500], bonusTiers: tiers, bonusMode: 'discount', currency: 'USD' },
    })
    const below = wrapper.get('[data-testid="quick-amount-50"]')
    const hit = wrapper.get('[data-testid="quick-amount-500"]')
    expect(below.find('[data-testid="quick-amount-bonus-badge"]').exists()).toBe(false)
    expect(below.get('[data-testid="quick-amount-credited"]').text()).toContain('50.00')
    expect(hit.get('[data-testid="quick-amount-bonus-badge"]').text()).toBe('30% OFF')
    // 500 × (1 − 30%) = 350
    expect(hit.get('[data-testid="quick-amount-credited"]').text()).toContain('350.00')
  })
})
