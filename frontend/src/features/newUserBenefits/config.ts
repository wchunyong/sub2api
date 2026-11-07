import { getNewUserRechargeBonus } from '@/api/activities'
import { newUserBenefitConfig } from './state'

export async function refreshNewUserBenefitConfig(): Promise<void> {
  try {
    newUserBenefitConfig.value = await getNewUserRechargeBonus()
  } catch {
    newUserBenefitConfig.value = null
  }
}
