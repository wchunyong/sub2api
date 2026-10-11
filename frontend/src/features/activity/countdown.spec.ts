import { describe, expect, it } from 'vitest'
import { activityNavigationVisible, countdownWithServerSkew, formatCountdown } from './countdown'

describe('ten yuan activity countdown', () => {
  it('uses the server clock when the browser is ahead by five minutes', () => {
    const receivedAt = Date.parse('2026-10-11T10:05:00+08:00')
    const serverNow = '2026-10-11T10:00:00+08:00'
    const target = '2026-10-11T10:00:12+08:00'
    expect(countdownWithServerSkew(target, serverNow, receivedAt, receivedAt)).toBe(12)
    expect(countdownWithServerSkew(target, serverNow, receivedAt, receivedAt + 12_000)).toBe(0)
  })

  it('displays days, hours, minutes and seconds', () => {
    expect(formatCountdown(7 * 86400 + 65)).toBe('07:00:01:05')
  })

  it('shows navigation for scheduled and active rounds until the exact end', () => {
    const endsAt = '2026-10-18T10:00:00+08:00'
    const end = new Date(endsAt).getTime()
    expect(activityNavigationVisible({ enabled: true, ends_at: endsAt }, end - 1)).toBe(true)
    expect(activityNavigationVisible({ enabled: true, ends_at: endsAt }, end)).toBe(false)
    expect(activityNavigationVisible({ enabled: false, ends_at: endsAt }, end - 1)).toBe(false)
  })
})
