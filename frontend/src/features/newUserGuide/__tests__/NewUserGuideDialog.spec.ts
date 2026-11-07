import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import NewUserGuideDialog from '../NewUserGuideDialog.vue'

vi.mock('vue-i18n', async () => ({
  ...(await vi.importActual<typeof import('vue-i18n')>('vue-i18n')),
  useI18n: () => ({ t: (key: string) => key }),
}))

const VIDEO = 'https://www.bilibili.com/video/BV1XgeB6UEQg'
const PDF = 'https://pub-17119e84a4e34086ab498f52e1943e3a.r2.dev/tutorial/win.pdf'

function mountDialog() {
  return mount(NewUserGuideDialog, {
    props: { videoUrl: VIDEO, pdfUrl: PDF },
    global: { stubs: { teleport: true } },
  })
}

describe('NewUserGuideDialog', () => {
  it('links the video and written guide cards to their URLs in a new tab', () => {
    const wrapper = mountDialog()
    const video = wrapper.get('[data-test="guide-video-link"]')
    const doc = wrapper.get('[data-test="guide-doc-link"]')
    expect(video.attributes('href')).toBe(VIDEO)
    expect(video.attributes('target')).toBe('_blank')
    expect(video.attributes('rel')).toContain('noopener')
    expect(doc.attributes('href')).toBe(PDF)
    expect(doc.attributes('target')).toBe('_blank')
  })

  it('emits close from the close button and Escape key', async () => {
    const wrapper = mountDialog()
    await wrapper.get('[data-test="close-guide"]').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(wrapper.emitted('close')).toHaveLength(2)
    wrapper.unmount()
  })

  it('copies the support WeChat ID', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    const wrapper = mountDialog()
    await wrapper.get('[data-test="copy-wechat"]').trigger('click')
    await Promise.resolve()
    expect(writeText).toHaveBeenCalledWith('lianjieai_top')
    expect(wrapper.get('[data-test="copy-wechat"]').text()).toBe('newUserGuide.copied')
  })
})
