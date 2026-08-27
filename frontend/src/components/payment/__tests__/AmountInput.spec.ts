import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import AmountInput from '../AmountInput.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => ({
      'payment.quickAmounts': '充值金额',
    }[key] ?? key),
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
