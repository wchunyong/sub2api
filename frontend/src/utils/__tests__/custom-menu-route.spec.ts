import { describe, expect, it } from 'vitest'
import { resolveCustomMenuRoute } from '../custom-menu-route'

const origin = 'https://lianjieai.top'

describe('custom menu campaign routes', () => {
  it.each([
    '/national-day-2026',
    '/national-day-2026/',
    'https://lianjieai.top/national-day-2026',
  ])('routes the local campaign directly: %s', (url) => {
    expect(resolveCustomMenuRoute({ url }, origin)).toBe('/national-day-2026')
  })

  it('preserves query parameters and fragments without adding embed credentials', () => {
    expect(resolveCustomMenuRoute({ url: '/national-day-2026?from=menu#benefits' }, origin))
      .toBe('/national-day-2026?from=menu#benefits')
  })

  it.each([
    'https://other.example/national-day-2026',
    'https://lianjieai.top.evil.example/national-day-2026',
    'http://lianjieai.top/national-day-2026',
    'https://user:password@lianjieai.top/national-day-2026',
    'javascript:alert(1)',
    'md:guide',
    'https://[invalid',
    '/custom/campaign',
    '/docs',
    '',
  ])('retains existing handling for other targets: %s', (url) => {
    expect(resolveCustomMenuRoute({ url }, origin)).toBeNull()
  })

  it('preserves Markdown page precedence', () => {
    expect(resolveCustomMenuRoute({ url: '/national-day-2026', page_slug: 'guide' }, origin)).toBeNull()
  })
})
