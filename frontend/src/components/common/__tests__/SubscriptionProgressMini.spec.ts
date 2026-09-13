import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import SubscriptionProgressMini from '../SubscriptionProgressMini.vue'

vi.mock('@/stores', () => ({
  useSubscriptionStore: () => ({
    activeSubscriptions: [
      {
        id: 1,
        group_id: 10,
        daily_usage_usd: 0,
        weekly_usage_usd: 0,
        monthly_usage_usd: 0,
        expires_at: '2026-09-15T00:00:00Z',
        group: {
          id: 10,
          name: 'Crazy Monday',
          daily_limit_usd: 10,
          weekly_limit_usd: null,
          monthly_limit_usd: null,
        },
      },
    ],
    hasActiveSubscriptions: true,
    fetchActiveSubscriptions: vi.fn().mockResolvedValue(undefined),
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) =>
        ({
          'subscriptionProgress.title': 'Free Quota',
          'subscriptionProgress.viewDetails': 'Free Quota',
          'subscriptionProgress.activeCount': '1 active subscription',
          'subscriptionProgress.daily': 'Daily',
          'subscriptionProgress.daysRemaining': '1 day left',
          'subscriptionProgress.viewAll': 'View all subscriptions',
        })[key] ?? key,
    }),
  }
})

describe('SubscriptionProgressMini', () => {
  it('labels the header quota popover as free quota without subscription count or view-all link', async () => {
    const wrapper = mount(SubscriptionProgressMini, {
      global: {
        stubs: {
          Icon: { props: ['name'], template: '<span>{{ name }}</span>' },
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })

    expect(wrapper.get('button').attributes('title')).toBe('Free Quota')

    await wrapper.get('button').trigger('click')

    expect(wrapper.text()).toContain('Free Quota')
    expect(wrapper.text()).not.toContain('1 active subscription')
    expect(wrapper.text()).not.toContain('View all subscriptions')
  })
})
