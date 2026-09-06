import { createServerFn } from '@tanstack/react-start'
import { z } from 'zod'
import { apiDelete, apiGet, apiPost, apiPut } from './api'
import { cleanRepoId } from './repo-id'
import type { Note, OpenResult, Overview, ProjectDetail, RepoDiff } from './types'

function repoURL(id: string, suffix = '') {
  return `/api/repos/${encodeURIComponent(cleanRepoId(id))}${suffix}`
}

export const getOverview = createServerFn({ method: 'GET' }).handler(async () => {
  return apiGet<Overview>('/api/overview')
})

export const rescan = createServerFn({ method: 'POST' }).handler(async () => {
  return apiPost<Overview>('/api/scan')
})

export const selectWorkspace = createServerFn({ method: 'POST' })
  .validator(z.object({ id: z.string().min(1) }))
  .handler(async ({ data }) => {
    return apiPost<Overview>(`/api/workspaces/${encodeURIComponent(data.id)}/select`)
  })

export const addWorkspace = createServerFn({ method: 'POST' })
  .validator(
    z.object({
      name: z.string().min(1),
      kind: z.string().optional(),
      roots: z.array(z.string()).optional(),
      host: z.string().optional(),
      url: z.string().optional(),
      listen: z.string().optional(),
    }),
  )
  .handler(async ({ data }) => {
    return apiPost<Overview>('/api/workspaces', data)
  })

export const addRoot = createServerFn({ method: 'POST' })
  .validator(z.object({ path: z.string().min(1) }))
  .handler(async ({ data }) => {
    return apiPost<Overview>('/api/roots', data)
  })

export const getProject = createServerFn({ method: 'GET' })
  .validator(z.object({ id: z.string().min(1) }))
  .handler(async ({ data }) => {
    return apiGet<ProjectDetail>(repoURL(data.id))
  })

export const openProject = createServerFn({ method: 'POST' })
  .validator(z.object({ id: z.string().min(1), action: z.string().min(1) }))
  .handler(async ({ data }) => {
    return apiPost<OpenResult>(repoURL(data.id, '/open'), {
      action: data.action,
    })
  })

export const setFocus = createServerFn({ method: 'POST' })
  .validator(
    z.object({
      id: z.string().min(1),
      action: z.string().min(1),
      hours: z.number().optional(),
    }),
  )
  .handler(async ({ data }) => {
    return apiPost<Overview>(repoURL(data.id, '/focus'), {
      action: data.action,
      hours: data.hours,
    })
  })

export const setNext = createServerFn({ method: 'POST' })
  .validator(z.object({ id: z.string().min(1), text: z.string() }))
  .handler(async ({ data }) => {
    return apiPost<Overview>(repoURL(data.id, '/next'), {
      text: data.text,
    })
  })

export const getDiff = createServerFn({ method: 'GET' })
  .validator(
    z.object({
      id: z.string().min(1),
      ref: z.string().min(1),
      path: z.string().optional(),
    }),
  )
  .handler(async ({ data }) => {
    const q = data.path ? `?path=${encodeURIComponent(data.path)}` : ''
    return apiGet<RepoDiff>(repoURL(data.id, `/diff/${encodeURIComponent(data.ref)}${q}`))
  })

export const getNote = createServerFn({ method: 'GET' })
  .validator(z.object({ id: z.string().min(1), name: z.string().min(1) }))
  .handler(async ({ data }) => {
    return apiGet<Note>(repoURL(data.id, `/notes/${encodeURIComponent(data.name)}`))
  })

export const createNote = createServerFn({ method: 'POST' })
  .validator(
    z.object({
      id: z.string().min(1),
      title: z.string().min(1),
      content: z.string().optional(),
    }),
  )
  .handler(async ({ data }) => {
    return apiPost<Note>(repoURL(data.id, '/notes'), {
      title: data.title,
      content: data.content,
    })
  })

export const saveNote = createServerFn({ method: 'POST' })
  .validator(
    z.object({
      id: z.string().min(1),
      name: z.string().min(1),
      content: z.string(),
    }),
  )
  .handler(async ({ data }) => {
    return apiPut<Note>(repoURL(data.id, `/notes/${encodeURIComponent(data.name)}`), {
      content: data.content,
    })
  })

export const deleteNote = createServerFn({ method: 'POST' })
  .validator(z.object({ id: z.string().min(1), name: z.string().min(1) }))
  .handler(async ({ data }) => {
    return apiDelete<{ ok: boolean }>(repoURL(data.id, `/notes/${encodeURIComponent(data.name)}`))
  })

export const runPlugin = createServerFn({ method: 'POST' })
  .validator(
    z.object({
      plugin: z.string().min(1),
      action: z.string().min(1),
      repo: z.string().min(1),
      harness: z.string().optional(),
      session: z.string().optional(),
      target: z.string().optional(),
      url: z.string().optional(),
    }),
  )
  .handler(async ({ data }) => {
    return apiPost<OpenResult>(`/api/plugins/${encodeURIComponent(data.plugin)}/run`, {
      action: data.action,
      repo: cleanRepoId(data.repo),
      harness: data.harness,
      session: data.session,
      target: data.target,
      url: data.url,
    })
  })
