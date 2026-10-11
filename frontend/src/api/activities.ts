import { apiClient } from './client'

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
