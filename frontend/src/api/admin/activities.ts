import { apiClient } from '../client'

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
