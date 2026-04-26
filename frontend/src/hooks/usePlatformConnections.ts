import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as api from '@/api/platformConnections'
import type { UpsertPlatformConnectionRequest } from '@/types'

export function usePlatformConnections() {
  return useQuery({
    queryKey: ['platform-connections'],
    queryFn: async () => {
      const res = await api.listPlatformConnections()
      return res.data.items
    },
  })
}

export function useUpsertPlatformConnection() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: UpsertPlatformConnectionRequest) => api.upsertPlatformConnection(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['platform-connections'] }),
  })
}

export function useDeletePlatformConnection() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ platform, platformNodeId }: { platform: string; platformNodeId: string }) =>
      api.deletePlatformConnection(platform, platformNodeId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['platform-connections'] }),
  })
}
