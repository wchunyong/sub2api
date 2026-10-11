import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import TenYuanActivityView from '../TenYuanActivityView.vue'

const getActivity = vi.hoisted(() => vi.fn())
vi.mock('@/api/activities', () => ({ getTenYuanActivity: getActivity }))
vi.mock('vue-i18n', async () => ({ ...(await vi.importActual<typeof import('vue-i18n')>('vue-i18n')), useI18n: () => ({ t: (key: string) => key }) }))

function mountPage() {
  return mount(TenYuanActivityView, {
    global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, RouterLink: { template: '<a><slot /></a>' } } },
  })
}

describe('user ten yuan activity', () => {
  beforeEach(() => getActivity.mockReset())

  it('shows a pre-start countdown and the real daily group limit', async () => {
    const now = Date.now()
    getActivity.mockResolvedValue({ enabled: true, status: 'scheduled', starts_at: new Date(now + 60_000).toISOString(), ends_at: new Date(now + 168 * 3600_000 + 60_000).toISOString(), server_now: new Date(now).toISOString(), claimed: false, daily_limit_usd: 8, min_recharge: 10 })
    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.get('[data-test="activity-countdown"]').text()).toMatch(/^00:00:(00:5[5-9]|01:00)$/)
    expect(wrapper.text()).toContain('8 USD')
    expect(wrapper.text()).toContain('tenYuanActivity.scheduled')
    expect(wrapper.find('a.btn-primary').exists()).toBe(false)
    wrapper.unmount()
  })

  it('shows the claimed state without a recharge action', async () => {
    const now = Date.now()
    getActivity.mockResolvedValue({ enabled: true, status: 'active', starts_at: new Date(now - 3600_000).toISOString(), ends_at: new Date(now + 3600_000).toISOString(), server_now: new Date(now).toISOString(), claimed: true, daily_limit_usd: 10, min_recharge: 10 })
    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.text()).toContain('tenYuanActivity.claimed')
    expect(wrapper.find('a.btn-primary').exists()).toBe(false)
    wrapper.unmount()
  })

  it('explains that an early closure preserves existing claims', async () => {
    const now = Date.now()
    getActivity.mockResolvedValue({ enabled: false, status: 'closed', starts_at: new Date(now - 3600_000).toISOString(), ends_at: new Date(now + 3600_000).toISOString(), server_now: new Date(now).toISOString(), claimed: true, min_recharge: 10 })
    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.text()).toContain('tenYuanActivity.closedClaimed')
    expect(wrapper.find('a.btn-primary').exists()).toBe(false)
    wrapper.unmount()
  })
})
