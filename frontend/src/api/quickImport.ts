import { apiClient } from './client'
import { buildApiUrl } from './url'
import type { ImportAgent } from '@/utils/quickImport'
export interface ImportTicket { ticket: string; expires_in: number; agent: ImportAgent; model: string }
export async function createImportTicket(keyId: number, agent: ImportAgent, model: string, signal?: AbortSignal): Promise<ImportTicket> {
  const { data } = await apiClient.post<ImportTicket>('/quick-import/tickets', { key_id: keyId, agent, model: model.trim() }, { signal })
  return data
}
export function importServer(): string {
  return new URL(buildApiUrl(''), window.location.origin).href.replace(/\/api\/v1\/?$/, '')
}
