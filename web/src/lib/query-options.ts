import { queryOptions } from '@tanstack/react-query'
import { getDiff, getNote, getOverview, getProject } from './server-functions'

export function overviewQueryOptions() {
  return queryOptions({
    queryKey: ['overview'],
    queryFn: () => getOverview(),
    staleTime: 10_000,
  })
}

export function projectQueryOptions(id: string) {
  return queryOptions({
    queryKey: ['project', id],
    queryFn: () => getProject({ data: { id } }),
    staleTime: 8_000,
  })
}

export function diffQueryOptions(id: string, ref: string, path?: string) {
  return queryOptions({
    queryKey: ['diff', id, ref, path ?? ''],
    queryFn: () => getDiff({ data: { id, ref, path } }),
    staleTime: 4_000,
  })
}

export function noteQueryOptions(id: string, name: string) {
  return queryOptions({
    queryKey: ['note', id, name],
    queryFn: () => getNote({ data: { id, name } }),
    staleTime: 4_000,
  })
}
