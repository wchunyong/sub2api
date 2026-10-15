export const NEW_USER_BENEFIT_DURATION_MS = 24 * 60 * 60 * 1000

export function isNewUserBenefitActive(createdAt: string | undefined, now = Date.now()): boolean {
  if (!createdAt) return false
  const created = Date.parse(createdAt)
  return Number.isFinite(created) && created <= now && now < created + NEW_USER_BENEFIT_DURATION_MS
}

export function newUserBenefitDismissalKey(userId: number): string {
  return `new-user-benefits-dismissed:${userId}`
}
