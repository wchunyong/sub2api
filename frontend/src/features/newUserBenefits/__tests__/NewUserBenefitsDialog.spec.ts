import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import NewUserBenefitsDialog from '../NewUserBenefitsDialog.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

afterEach(() => vi.useRealTimers())

describe('new user benefits dialog', () => {
  it('reveals its close button only after three seconds', async () => {
    vi.useFakeTimers()
    const wrapper = mount(NewUserBenefitsDialog, {
      global: { stubs: { teleport: true, NewUserBenefitsContent: true } },
    })
    expect(wrapper.find('[data-test="close-benefits"]').exists()).toBe(false)
    await vi.advanceTimersByTimeAsync(2999)
    expect(wrapper.find('[data-test="close-benefits"]').exists()).toBe(false)
    await vi.advanceTimersByTimeAsync(1)
    expect(wrapper.find('[data-test="close-benefits"]').exists()).toBe(true)
    await wrapper.get('[data-test="close-benefits"]').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
    wrapper.unmount()
  })
})
