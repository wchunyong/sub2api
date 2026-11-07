import { ref } from 'vue'
import type { NewUserRechargeBonusConfig } from '@/types/payment'

export const DEFAULT_NEW_USER_BONUS_AMOUNT = 5
export const DEFAULT_NEW_USER_MIN_AMOUNT = 10

/** 管理员在活动页配置的新用户权益；null 表示尚未加载或加载失败，视为关闭。 */
export const newUserBenefitConfig = ref<NewUserRechargeBonusConfig | null>(null)
