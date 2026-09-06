import { z } from 'zod'

const envSchema = z.object({
  DEVDASH_API_URL: z.string().default('http://127.0.0.1:8789'),
})

export class DevdashApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'DevdashApiError'
    this.status = status
  }
}

function apiBase() {
  return envSchema.parse({
    DEVDASH_API_URL: process.env.DEVDASH_API_URL ?? 'http://127.0.0.1:8789',
  }).DEVDASH_API_URL
}

export async function apiGet<T>(path: string): Promise<T> {
  return apiJSON<T>(path, { method: 'GET' })
}

export async function apiPost<T>(path: string, body?: unknown): Promise<T> {
  return apiJSON<T>(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
}

export async function apiPut<T>(path: string, body?: unknown): Promise<T> {
  return apiJSON<T>(path, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
}

export async function apiDelete<T>(path: string): Promise<T> {
  return apiJSON<T>(path, { method: 'DELETE' })
}

async function apiJSON<T>(path: string, init: RequestInit): Promise<T> {
  const response = await fetch(new URL(path, apiBase()), {
    ...init,
    cache: 'no-store',
    headers: {
      Accept: 'application/json',
      ...(init.headers ?? {}),
    },
  })
  if (!response.ok) {
    let message = `devdash api ${response.status}`
    try {
      const payload = (await response.json()) as { error?: string }
      if (payload.error) message = payload.error
    } catch {
      // keep fallback
    }
    throw new DevdashApiError(message, response.status)
  }
  return (await response.json()) as T
}
