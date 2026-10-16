import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import ContactSupportDialog from '../ContactSupportDialog.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => ({
      'nav.contactUs': '联系我们',
      'nav.contactSupportHint': '有任何问题，请联系我们获得技术支持。'
    })[key] ?? key
  })
}))

enableAutoUnmount(afterEach)

describe('ContactSupportDialog', () => {
  it('shows technical support contacts and closes on request', async () => {
    const wrapper = mount(ContactSupportDialog, {
      attachTo: document.body,
      props: { show: true },
      global: { stubs: { Icon: true } }
    })

    const dialog = document.body.querySelector('[role="dialog"]')
    expect(dialog?.textContent).toContain('联系我们')
    expect(dialog?.textContent).toContain('有任何问题，请联系我们获得技术支持。')
    expect(dialog?.textContent).toContain('TG:')
    expect(dialog?.textContent).toContain('QQ: 3505152117')
    expect(dialog?.textContent).toContain('WX: lianjieai_top')
    const tg = dialog?.querySelector<HTMLAnchorElement>('a[href="https://t.me/sfv_123"]')
    expect(tg?.textContent).toBe('https://t.me/sfv_123')
    expect(tg?.getAttribute('target')).toBe('_blank')
    expect(tg?.getAttribute('rel')).toContain('noopener')

    dialog?.querySelector('button')?.click()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
