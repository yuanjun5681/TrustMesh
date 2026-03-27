import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as agentsApi from '@/api/agents'
import type { CreateAgentRequest, UpdateAgentRequest } from '@/types'

export function useAgents() {
  return useQuery({
    queryKey: ['agents'],
    queryFn: async () => {
      const res = await agentsApi.listAgents()
      return res.data.items
    },
  })
}

export function useAgent(id: string | undefined) {
  return useQuery({
    queryKey: ['agents', id],
    queryFn: async () => {
      const res = await agentsApi.getAgent(id!)
      return res.data
    },
    enabled: !!id,
  })
}

export function useCreateAgent() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateAgentRequest) => agentsApi.createAgent(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents'] }),
  })
}

export function useUpdateAgent() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateAgentRequest }) =>
      agentsApi.updateAgent(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents'] }),
  })
}

export function useAgentTasks(id: string | undefined, status?: string) {
  return useQuery({
    queryKey: ['agents', id, 'tasks', status ?? 'all'],
    queryFn: async () => {
      const res = await agentsApi.listAgentTasks(id!, status)
      return res.data.items
    },
    enabled: !!id,
    staleTime: 30_000,
  })
}

export function useAgentStats(id: string | undefined) {
  return useQuery({
    queryKey: ['agents', id, 'stats'],
    queryFn: async () => {
      const res = await agentsApi.getAgentStats(id!)
      return res.data
    },
    enabled: !!id,
    staleTime: 30_000,
  })
}

export function useDeleteAgent() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => agentsApi.deleteAgent(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents'] }),
  })
}
