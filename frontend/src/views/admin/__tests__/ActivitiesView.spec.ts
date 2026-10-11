import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ActivitiesView from '../ActivitiesView.vue'

const getActivity = vi.hoisted(() => vi.fn())
const scheduleActivity = vi.hoisted(() => vi.fn())
const closeActivity = vi.hoisted(() => vi.fn())
const getGroups = vi.hoisted(() => vi.fn())

vi.mock('@/api/admin/activities', () => ({
  getTenYuanActivityAdmin: getActivity,
  scheduleTenYuanActivity: scheduleActivity,
  closeTenYuanActivity: closeActivity,
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
  })

  it('selects the exact default group and sends Beijing time to the second', async () => {
    const wrapper = mountPage()
    await flushPromises()
    expect((wrapper.get('[data-test="activity-group"]').element as HTMLSelectElement).value).toBe('10')
    await wrapper.get('[data-test="activity-start"]').setValue('2026-10-12T09:30:45')
    await wrapper.get('button.btn-primary').trigger('click')
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
    expect((wrapper.get('button.btn-primary').element as HTMLButtonElement).disabled).toBe(true)
    await wrapper.get('[data-test="activity-group"]').setValue('11')
    expect((wrapper.get('button.btn-primary').element as HTMLButtonElement).disabled).toBe(false)
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
})
