import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

import AnnouncementBanner from '../AnnouncementBanner.vue'
import { useAnnouncementStore } from '@/stores/announcements'
import { useAuthStore } from '@/stores/auth'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

describe('AnnouncementBanner', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    document.body.innerHTML = ''
    document.documentElement.classList.remove('has-announcement-banner')
    document.documentElement.style.removeProperty('--announcement-banner-height')
    document.documentElement.style.removeProperty('--announcement-banner-offset')
    vi.restoreAllMocks()
  })

  it('renders the first unread banner announcement with its CTA', () => {
    const store = useAnnouncementStore()
    store.announcements = [
      {
        id: 10,
        title: 'Banner',
        content: '💪 2024 年度「SaaS 产品用户体验白皮书」正式发布，体验驱动增长！',
        notify_mode: 'banner',
        banner_config: {
          whole_banner_click_enabled: true,
          click_url: 'https://hovxm.com',
          button_enabled: true,
          button_text: '了解详情',
          button_url: 'https://hovxm.com/docs',
        },
        created_at: '2026-07-24T07:30:00Z',
        updated_at: '2026-07-24T07:30:00Z',
      },
    ]

    const wrapper = mount(AnnouncementBanner)

    expect(wrapper.text()).toContain('2024 年度')
    expect(wrapper.text()).toContain('了解详情')
  })

  it('participates in normal page layout instead of covering the top bar', () => {
    const store = useAnnouncementStore()
    store.announcements = [
      {
        id: 12,
        title: 'Inline banner',
        content: '插入式横幅',
        notify_mode: 'banner',
        banner_config: {},
        created_at: '2026-07-24T07:30:00Z',
        updated_at: '2026-07-24T07:30:00Z',
      },
    ]

    const wrapper = mount(AnnouncementBanner)
    const banner = wrapper.get('[data-testid="announcement-banner"]')

    expect(banner.classes()).not.toContain('fixed')
    expect(banner.classes()).not.toContain('top-0')
    expect(banner.classes()).toContain('z-[50]')
    expect(banner.classes()).toEqual(
      expect.arrayContaining(['from-violet-600', 'via-rose-500', 'to-amber-400'])
    )
  })

  it('sets a global banner offset while visible so fixed chrome can move down', async () => {
    const store = useAnnouncementStore()
    store.announcements = [
      {
        id: 14,
        title: 'Offset banner',
        content: '全站下移',
        notify_mode: 'banner',
        banner_config: {},
        created_at: '2026-07-24T07:30:00Z',
        updated_at: '2026-07-24T07:30:00Z',
      },
    ]

    const wrapper = mount(AnnouncementBanner)
    await nextTick()
    await nextTick()

    expect(document.documentElement.classList.contains('has-announcement-banner')).toBe(true)
    expect(document.documentElement.style.getPropertyValue('--announcement-banner-height')).not.toBe('')
    expect(document.documentElement.style.getPropertyValue('--announcement-banner-offset')).not.toBe('')

    wrapper.unmount()

    expect(document.documentElement.classList.contains('has-announcement-banner')).toBe(false)
  })

  it('reduces the fixed chrome offset as the banner scrolls out of view', async () => {
    const store = useAnnouncementStore()
    store.announcements = [
      {
        id: 15,
        title: 'Scrolling banner',
        content: '滚动时整体上移',
        notify_mode: 'banner',
        banner_config: {},
        created_at: '2026-07-24T07:30:00Z',
        updated_at: '2026-07-24T07:30:00Z',
      },
    ]

    mount(AnnouncementBanner)
    await nextTick()
    await nextTick()

    expect(document.documentElement.style.getPropertyValue('--announcement-banner-offset')).toBe('48px')

    vi.spyOn(window, 'scrollY', 'get').mockReturnValue(60)
    window.dispatchEvent(new Event('scroll'))

    expect(document.documentElement.style.getPropertyValue('--announcement-banner-offset')).toBe('0px')
  })

  it('marks the banner read when dismissed', async () => {
    const authStore = useAuthStore()
    authStore.user = {
      id: 1,
      username: 'user',
      email: 'user@example.com',
      role: 'user',
      balance: 0,
      concurrency: 1,
      status: 'active',
      allowed_groups: null,
      balance_notify_enabled: false,
      balance_notify_threshold: null,
      balance_notify_extra_emails: [],
      created_at: '2026-07-24T07:30:00Z',
      updated_at: '2026-07-24T07:30:00Z',
    }
    authStore.token = 'token'
    const store = useAnnouncementStore()
    store.announcements = [
      {
        id: 11,
        title: 'Dismissible banner',
        content: '横幅内容',
        notify_mode: 'banner',
        banner_config: {},
        created_at: '2026-07-24T07:30:00Z',
        updated_at: '2026-07-24T07:30:00Z',
      },
    ]
    const markAsRead = vi.spyOn(store, 'markAsRead').mockResolvedValue()

    const wrapper = mount(AnnouncementBanner)
    await wrapper.get('[data-testid="announcement-banner-dismiss"]').trigger('click')

    expect(markAsRead).toHaveBeenCalledWith(11)
  })

  it('lets anonymous visitors close the banner only for the current page session', async () => {
    const store = useAnnouncementStore()
    store.announcements = [
      {
        id: 13,
        title: 'Guest banner',
        content: '访客横幅',
        notify_mode: 'banner',
        banner_config: {},
        created_at: '2026-07-24T07:30:00Z',
        updated_at: '2026-07-24T07:30:00Z',
      },
    ]
    const markAsRead = vi.spyOn(store, 'markAsRead').mockResolvedValue()

    const wrapper = mount(AnnouncementBanner)
    await wrapper.get('[data-testid="announcement-banner-dismiss"]').trigger('click')

    expect(markAsRead).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="announcement-banner"]').exists()).toBe(false)

    wrapper.unmount()
    const remounted = mount(AnnouncementBanner)

    expect(remounted.find('[data-testid="announcement-banner"]').exists()).toBe(true)
  })
})
