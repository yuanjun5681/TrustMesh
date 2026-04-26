import { api } from './client'
import type { ApiListResponse, ApiResponse, PlatformConnection, UpsertPlatformConnectionRequest } from '@/types'

export async function listPlatformConnections() {
  return api.get('platform-connections').json<ApiListResponse<PlatformConnection>>()
}

export async function upsertPlatformConnection(input: UpsertPlatformConnectionRequest) {
  return api.post('platform-connections', { json: input }).json<ApiResponse<PlatformConnection>>()
}

export async function deletePlatformConnection(platform: string, platformNodeId: string) {
  await api.delete(`platform-connections/${platform}/${platformNodeId}`)
}
