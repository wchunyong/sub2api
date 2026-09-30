import { enableAutoUnmount, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import SubscriptionProgressMini from '../SubscriptionProgressMini.vue'

const store = vi.hoisted(() => ({
  activeSubscriptions: [] as unknown[],
  hasActiveSubscriptions: true,
  fetchActiveSubscriptions: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('@/stores', () => ({
  useSubscriptionStore: () => store,
}))

vi.mock('@/utils/featureFlags', () => ({
  FeatureFlags: { subscription: 'subscription' },
  isFeatureFlagEnabled: () => true,
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
          'subscriptionProgress.expiresToday': 'subscriptionProgress.expiresToday',
          'subscriptionProgress.expiresTomorrow': 'subscriptionProgress.expiresTomorrow',
          'subscriptionProgress.expired': 'subscriptionProgress.expired',
        })[key] ?? key,
    }),
  }
})

enableAutoUnmount(afterEach)

beforeEach(() => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date(2026, 8, 22, 12))
  store.activeSubscriptions = [
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
  ]
  store.hasActiveSubscriptions = true
  store.fetchActiveSubscriptions.mockClear()
})

afterEach(() => vi.useRealTimers())

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

  it.each([
    [new Date(2026, 8, 22, 18), 'expiresToday'],
    [new Date(2026, 8, 23, 18), 'expiresTomorrow'],
    [new Date(2026, 8, 22, 12), 'expired'],
    [new Date(2026, 8, 25, 12), 'daysRemaining'],
  ])('labels %s as %s', async (expires, label) => {
    store.activeSubscriptions = [
      { id: 1, group_id: 1, expires_at: expires.toISOString(), group: { name: 'Plan' } },
    ]
    const wrapper = mount(SubscriptionProgressMini, {
      global: { stubs: { Icon: true, RouterLink: true } },
    })

    await wrapper.get('button').trigger('click')

    expect(wrapper.text()).toContain(
      label === 'daysRemaining' ? '1 day left' : 'subscriptionProgress.' + label,
    )
  })
})
