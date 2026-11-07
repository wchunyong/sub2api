import { apiClient } from '../client'
import type { ProgrammersDayState } from '../activities'
import type { NewUserGuideConfig, NewUserRechargeBonusConfig } from '@/types/payment'

export interface TenYuanActivityRun {
  id: number
  group_id: number
  starts_at: string
  ends_at: string
  disabled_at?: string
  status: 'scheduled' | 'active' | 'closed' | 'ended'
}

export interface TenYuanActivityAdminState {
  current: TenYuanActivityRun | null
  recent: TenYuanActivityRun[]
  default_group_id: number
  server_now: string
}

const path = '/admin/payment/activities/ten-yuan-seven-days'

export async function getTenYuanActivityAdmin(): Promise<TenYuanActivityAdminState> {
  const { data } = await apiClient.get<TenYuanActivityAdminState>(path)
  return data
}

export async function scheduleTenYuanActivity(groupId: number, startsAt: string): Promise<TenYuanActivityRun> {
  const { data } = await apiClient.put<TenYuanActivityRun>(path, { group_id: groupId, starts_at: startsAt })
  return data
}

export async function closeTenYuanActivity(runId: number): Promise<void> {
  await apiClient.post(`${path}/close`, { run_id: runId })
}

const programmersDayPath = '/admin/payment/activities/programmers-day'

export async function getProgrammersDayAdmin(): Promise<ProgrammersDayState> {
  const { data } = await apiClient.get<ProgrammersDayState>(programmersDayPath)
  return data
}

export async function updateProgrammersDay(enabled: boolean, bonusPercent: number): Promise<ProgrammersDayState> {
  const { data } = await apiClient.put<ProgrammersDayState>(programmersDayPath, { enabled, bonus_percent: bonusPercent })
  return data
}

const newUserRechargeBonusPath = '/admin/payment/activities/new-user-recharge-bonus'

export async function getNewUserRechargeBonusAdmin(): Promise<NewUserRechargeBonusConfig> {
  const { data } = await apiClient.get<NewUserRechargeBonusConfig>(newUserRechargeBonusPath)
  return data
}

const newUserGuidePath = '/admin/payment/activities/new-user-guide'

export async function getNewUserGuideAdmin(): Promise<NewUserGuideConfig> {
  const { data } = await apiClient.get<NewUserGuideConfig>(newUserGuidePath)
  return data
}

export async function updateNewUserGuide(config: NewUserGuideConfig): Promise<NewUserGuideConfig> {
  const { data } = await apiClient.put<NewUserGuideConfig>(newUserGuidePath, config)
  return data
}

export async function updateNewUserRechargeBonus(config: NewUserRechargeBonusConfig): Promise<NewUserRechargeBonusConfig> {
  const { data } = await apiClient.put<NewUserRechargeBonusConfig>(newUserRechargeBonusPath, config)
  return data
}
