import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import AnnouncementBanner from '../AnnouncementBanner.vue'
import { useAnnouncementStore } from '@/stores/announcements'

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

    mount(AnnouncementBanner)

    expect(document.body.textContent).toContain('2024 年度')
    expect(document.body.textContent).toContain('了解详情')
  })

  it('marks the banner read when dismissed', async () => {
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

    mount(AnnouncementBanner)
    document.body.querySelector<HTMLButtonElement>('[data-testid="announcement-banner-dismiss"]')?.click()
    await Promise.resolve()

    expect(markAsRead).toHaveBeenCalledWith(11)
  })
})
