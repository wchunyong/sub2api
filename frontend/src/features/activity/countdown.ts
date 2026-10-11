export function countdownWithServerSkew(target: string, serverNow: string, clientReceivedAt: number, clientNow: number): number {
  const serverElapsedNow = new Date(serverNow).getTime() + (clientNow - clientReceivedAt)
  return Math.max(0, Math.ceil((new Date(target).getTime() - serverElapsedNow) / 1000))
}

export function formatCountdown(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const rest = seconds % 60
  return [days, hours, minutes, rest].map(value => String(value).padStart(2, '0')).join(':')
}

export function activityNavigationVisible(activity: { enabled: boolean; ends_at?: string } | null, serverNowMs: number): boolean {
  return !!activity?.enabled && !!activity.ends_at && serverNowMs < new Date(activity.ends_at).getTime()
}
