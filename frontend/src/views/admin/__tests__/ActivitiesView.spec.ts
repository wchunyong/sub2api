import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ActivitiesView from '../ActivitiesView.vue'

const getActivity = vi.hoisted(() => vi.fn())
const scheduleActivity = vi.hoisted(() => vi.fn())
const closeActivity = vi.hoisted(() => vi.fn())
const getGroups = vi.hoisted(() => vi.fn())
const getProgrammersDay = vi.hoisted(() => vi.fn())
const updateProgrammersDay = vi.hoisted(() => vi.fn())
const getNewUserBenefit = vi.hoisted(() => vi.fn())
const updateNewUserBenefit = vi.hoisted(() => vi.fn())
const getNewUserGuide = vi.hoisted(() => vi.fn())
const updateNewUserGuide = vi.hoisted(() => vi.fn())

vi.mock('@/api/admin/activities', () => ({
  getNewUserGuideAdmin: getNewUserGuide,
  updateNewUserGuide,
  getTenYuanActivityAdmin: getActivity,
  scheduleTenYuanActivity: scheduleActivity,
  closeTenYuanActivity: closeActivity,
  getProgrammersDayAdmin: getProgrammersDay,
  updateProgrammersDay,
  getNewUserRechargeBonusAdmin: getNewUserBenefit,
  updateNewUserRechargeBonus: updateNewUserBenefit,
}))
vi.mock('@/api/admin/groups', async () => ({ ...(await vi.importActual<typeof import('@/api/admin/groups')>('@/api/admin/groups')), getAll: getGroups }))
vi.mock('vue-i18n', async () => ({ ...(await vi.importActual<typeof import('vue-i18n')>('vue-i18n')), useI18n: () => ({ t: (key: string) => key }) }))

function mountPage() {
  return mount(ActivitiesView, {
    global: { stubs: { AppLayout: { template: '<div><slot /></div>' } } },
  })
}

describe('admin ten yuan activity', () => {
  beforeEach(() => {
    getActivity.mockReset().mockResolvedValue({ current: null, recent: [], default_group_id: 10, server_now: '2026-10-11T00:00:00Z' })
    getGroups.mockReset().mockResolvedValue([
      { id: 10, name: '十元七天乐', status: 'active', subscription_type: 'subscription', daily_limit_usd: 10 },
      { id: 11, name: 'Other', status: 'active', subscription_type: 'subscription', daily_limit_usd: 5 },
    ])
    scheduleActivity.mockReset().mockResolvedValue({})
    closeActivity.mockReset().mockResolvedValue(undefined)
    getProgrammersDay.mockReset().mockResolvedValue({ enabled: true, active: false, bonus_percent: 24, starts_at: '2026-10-24T00:00:00+08:00', ends_at: '2026-10-27T00:00:00+08:00' })
    updateProgrammersDay.mockReset().mockResolvedValue({ enabled: false, active: false, bonus_percent: 30, starts_at: '2026-10-24T00:00:00+08:00', ends_at: '2026-10-27T00:00:00+08:00' })
    getNewUserBenefit.mockReset().mockResolvedValue({ enabled: false, bonus_amount: 5, min_amount: 10 })
    updateNewUserBenefit.mockReset().mockImplementation(async (config: unknown) => config)
    getNewUserGuide.mockReset().mockResolvedValue({ video_url: 'https://video.example/a', pdf_url: 'https://pdf.example/a.pdf' })
    updateNewUserGuide.mockReset().mockImplementation(async (config: unknown) => config)
  })

  it('loads and saves the new user guide links', async () => {
    const wrapper = mountPage()
    await flushPromises()
    expect((wrapper.get('[data-test="new-user-guide-video"]').element as HTMLInputElement).value).toBe('https://video.example/a')
    await wrapper.get('[data-test="new-user-guide-pdf"]').setValue('https://pdf.example/b.pdf')
    await wrapper.get('[data-test="new-user-guide-save"]').trigger('click')
    await flushPromises()
    expect(updateNewUserGuide).toHaveBeenCalledWith({ video_url: 'https://video.example/a', pdf_url: 'https://pdf.example/b.pdf' })
  })

  it('loads and saves the new user first recharge benefit switch and amounts', async () => {
    const wrapper = mountPage()
    await flushPromises()
    const enabled = wrapper.get('[data-test="new-user-benefit-enabled"]').element as HTMLInputElement
    expect(enabled.checked).toBe(false)
    await wrapper.get('[data-test="new-user-benefit-enabled"]').setValue(true)
    await wrapper.get('[data-test="new-user-benefit-amount"]').setValue(8.5)
    await wrapper.get('[data-test="new-user-benefit-min"]').setValue(30)
    await wrapper.get('[data-test="new-user-benefit-save"]').trigger('click')
    await flushPromises()
    expect(updateNewUserBenefit).toHaveBeenCalledWith({ enabled: true, bonus_amount: 8.5, min_amount: 30 })
  })

  it('blocks saving invalid new user benefit amounts', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await wrapper.get('[data-test="new-user-benefit-amount"]').setValue(0)
    expect((wrapper.get('[data-test="new-user-benefit-save"]').element as HTMLButtonElement).disabled).toBe(true)
  })

  it('selects the exact default group and sends Beijing time to the second', async () => {
    const wrapper = mountPage()
    await flushPromises()
    expect((wrapper.get('[data-test="activity-group"]').element as HTMLSelectElement).value).toBe('10')
    await wrapper.get('[data-test="activity-start"]').setValue('2026-10-12T09:30:45')
    await wrapper.get('[data-test="ten-yuan-save"]').trigger('click')
    await flushPromises()
    expect(scheduleActivity).toHaveBeenCalledWith(10, '2026-10-12T09:30:45+08:00')
  })

  it('locks edits during an active round and closes the selected round', async () => {
    getActivity.mockResolvedValue({
      current: { id: 7, group_id: 10, starts_at: '2026-10-11T09:00:00+08:00', ends_at: '2026-10-18T09:00:00+08:00', status: 'active' },
      recent: [], default_group_id: 10, server_now: '2026-10-11T10:00:00+08:00',
    })
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(true)
    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.find('[data-test="activity-start"]').exists()).toBe(false)
    await wrapper.get('section:last-of-type button').trigger('click')
    await flushPromises()
    expect(closeActivity).toHaveBeenCalledWith(7)
    confirm.mockRestore()
  })

  it('requires a manual choice when the exact default group is unavailable', async () => {
    getActivity.mockResolvedValue({ current: null, recent: [], default_group_id: 0, server_now: '2026-10-11T00:00:00Z' })
    getGroups.mockResolvedValue([{ id: 11, name: 'Other', status: 'active', subscription_type: 'subscription', daily_limit_usd: 5 }])
    const wrapper = mountPage()
    await flushPromises()
    expect((wrapper.get('[data-test="activity-group"]').element as HTMLSelectElement).value).toBe('0')
    expect((wrapper.get('[data-test="ten-yuan-save"]').element as HTMLButtonElement).disabled).toBe(true)
    await wrapper.get('[data-test="activity-group"]').setValue('11')
    expect((wrapper.get('[data-test="ten-yuan-save"]').element as HTMLButtonElement).disabled).toBe(false)
  })

  it('uses the exact default group for a new round after an older round ends', async () => {
    getActivity.mockResolvedValue({
      current: { id: 7, group_id: 11, starts_at: '2026-10-01T09:00:00+08:00', ends_at: '2026-10-08T09:00:00+08:00', status: 'ended' },
      recent: [], default_group_id: 10, server_now: '2026-10-11T00:00:00Z',
    })
    const wrapper = mountPage()
    await flushPromises()
    expect((wrapper.get('[data-test="activity-group"]').element as HTMLSelectElement).value).toBe('10')
  })

  it('saves the annual 1024 switch and bonus percentage', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await wrapper.get('[data-test="programmers-day-enabled"]').setValue(false)
    await wrapper.get('[data-test="programmers-day-percent"]').setValue('30')
    await wrapper.get('[data-test="programmers-day-settings"] button').trigger('click')
    await flushPromises()
    expect(updateProgrammersDay).toHaveBeenCalledWith(false, 30)
  })
})
