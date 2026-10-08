import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AuthLayout from '@/components/layout/AuthLayout.vue'

const { fetchPublicSettingsMock } = vi.hoisted(() => ({
  fetchPublicSettingsMock: vi.fn(() => new Promise(() => undefined))
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    siteName: '连接AI',
    siteLogo: '',
    cachedPublicSettings: null,
    publicSettingsLoaded: false,
    fetchPublicSettings: fetchPublicSettingsMock
  })
}))

describe('AuthLayout branding', () => {
  it('shows the product brand while public settings are delayed', () => {
    const wrapper = mount(AuthLayout)

    expect(wrapper.get('h1').text()).toBe('连接AI')
    expect(wrapper.text()).toContain('lianjieai')
    expect(wrapper.find('img').exists()).toBe(false)
  })
})
