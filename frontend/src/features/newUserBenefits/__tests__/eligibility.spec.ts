import { describe, expect, it } from 'vitest'
import { isNewUserBenefitActive } from '../eligibility'

describe('new user benefit period', () => {
  const createdAt = '2026-10-15T00:00:00.000Z'

  it('is active until exactly 24 hours after account creation', () => {
    expect(isNewUserBenefitActive(createdAt, Date.parse(createdAt) + 24 * 60 * 60 * 1000 - 1)).toBe(true)
    expect(isNewUserBenefitActive(createdAt, Date.parse(createdAt) + 24 * 60 * 60 * 1000)).toBe(false)
  })

  it('rejects invalid or future creation timestamps', () => {
    expect(isNewUserBenefitActive('', Date.now())).toBe(false)
    expect(isNewUserBenefitActive('invalid', Date.now())).toBe(false)
    expect(isNewUserBenefitActive(createdAt, Date.parse(createdAt) - 1)).toBe(false)
  })
})
