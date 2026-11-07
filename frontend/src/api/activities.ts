import { apiClient } from './client'
import type { NewUserGuideConfig, NewUserRechargeBonusConfig } from '@/types/payment'

export interface TenYuanActivityState {
  enabled: boolean
  status: 'off' | 'scheduled' | 'active' | 'closed' | 'ended'
  run_id?: number
  starts_at?: string
  ends_at?: string
  server_now: string
  claimed: boolean
  group_name?: string
  daily_limit_usd?: number
  min_recharge: number
}

export async function getTenYuanActivity(): Promise<TenYuanActivityState> {
  const { data } = await apiClient.get<TenYuanActivityState>('/payment/activities/ten-yuan-seven-days')
  return data
}

export interface ProgrammersDayState {
  enabled: boolean
  active: boolean
  visible: boolean
  bonus_percent: number
  starts_at: string
  ends_at: string
  server_now: string
}

export async function getProgrammersDay(): Promise<ProgrammersDayState> {
  const { data } = await apiClient.get<ProgrammersDayState>('/payment/activities/programmers-day')
  return data
}

export async function getNewUserGuide(): Promise<NewUserGuideConfig> {
  const { data } = await apiClient.get<NewUserGuideConfig>('/payment/activities/new-user-guide')
  return data
}

export async function getNewUserRechargeBonus(): Promise<NewUserRechargeBonusConfig> {
  const { data } = await apiClient.get<NewUserRechargeBonusConfig>('/payment/activities/new-user-recharge-bonus')
  return data
}
