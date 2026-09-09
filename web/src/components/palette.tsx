import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import { useEffect, useMemo, useRef, useState } from 'react'
import { overviewQueryOptions } from '#/lib/query-options'
import { addRoot, openProject, rescan, runPlugin, selectWorkspace, setFocus } from '#/lib/server-functions'
import type { PaletteItem } from '#/lib/types'

export function CommandPalette() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const repoId = pathname.startsWith('/repos/') ? decodeURIComponent(pathname.slice('/repos/'.length)) : ''
  const { data } = useQuery(overviewQueryOptions())
  const [open, setOpen] = useState(false)
  const [q, setQ] = useState('')
  const [idx, setIdx] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)

  const items = useMemo(() => filterPalette(data?.palette ?? [], q, repoId), [data?.palette, q, repoId])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      const tag = (e.target as HTMLElement | null)?.tagName
      const typing = tag === 'INPUT' || tag === 'TEXTAREA'
      if ((e.key === ':' || (e.key === ' ' && !typing)) && !e.metaKey && !e.ctrlKey && !e.altKey) {
        if (typing && e.key === ' ') return
        e.preventDefault()
        setOpen(true)
        setQ('')
        setIdx(0)
      }
      if (e.key === 'Escape' && open) {
        e.preventDefault()
        setOpen(false)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open])

  useEffect(() => {
    if (open) inputRef.current?.focus()
  }, [open])

  if (!open) return null

  async function run(item: PaletteItem) {
    if (item.action === 'rescan') {
      const ov = await rescan()
      queryClient.setQueryData(['overview'], ov)
      setOpen(false)
      return
    }
    if (item.action === 'add-root') {
      const path = q.replace(/^add\s+root\s+/i, '').trim()
      if (!path.startsWith('/')) return
      const ov = await addRoot({ data: { path } })
      queryClient.setQueryData(['overview'], ov)
      setOpen(false)
      return
    }
    if (item.kind === 'jump' && item.repo_id) {
      setOpen(false)
      await navigate({ to: '/repos/$repoId', params: { repoId: item.repo_id } })
      return
    }
    if ((item.action === 'editor' || item.action === 'term' || item.action === 'url') && item.repo_id) {
      await openProject({ data: { id: item.repo_id, action: item.action } })
      setOpen(false)
      return
    }
    if ((item.action === 'pin' || item.action === 'snooze' || item.action === 'archive') && item.repo_id) {
      const ov = await setFocus({ data: { id: item.repo_id, action: item.action } })
      queryClient.setQueryData(['overview'], ov)
      setOpen(false)
      return
    }
    if (item.kind === 'workspace' || item.action === 'workspace') {
      const id = item.id.replace(/^ws:/, '')
      const ov = await selectWorkspace({ data: { id } })
      queryClient.setQueryData(['overview'], ov)
      queryClient.setQueryData(['workspaces'], {
        active: ov.workspace?.id,
        workspaces: ov.workspaces ?? [],
      })
      queryClient.removeQueries({ queryKey: ['project'] })
      setOpen(false)
      await navigate({ to: '/' })
      return
    }
    if (item.kind === 'plugin') {
      const target = item.repo_id || repoId || data?.today?.[0]?.repo.id
      if (!target) return
      const plugin = item.plugin || 'acc'
      let action = item.action || 'launch'
      if (action === 'acc:launch') action = 'launch'
      await runPlugin({
        data: {
          plugin,
          action,
          repo: target,
          harness: plugin === 'acc' ? 'claude' : undefined,
        },
      })
      setOpen(false)
    }
  }

  return (
    <div className="palette" role="dialog" aria-label="command palette">
      <p className="muted">palette  : or space  esc close</p>
      <input
        ref={inputRef}
        aria-label="command"
        placeholder="jump · editor · tmux · rescan · add root /path"
        value={q}
        onChange={(e) => {
          setQ(e.target.value)
          setIdx(0)
        }}
        onKeyDown={(e) => {
          if (e.key === 'Escape') setOpen(false)
          if (e.key === 'ArrowDown') {
            e.preventDefault()
            setIdx((i) => Math.min(items.length - 1, i + 1))
          }
          if (e.key === 'ArrowUp') {
            e.preventDefault()
            setIdx((i) => Math.max(0, i - 1))
          }
          if (e.key === 'Enter' && items[idx]) {
            e.preventDefault()
            void run(items[idx])
          }
        }}
      />
      <div className="palette-list">
        {items.length === 0 ? (
          <p className="empty muted">no matches</p>
        ) : (
          items.map((item, i) => (
            <button
              key={item.id}
              className={i === idx ? 'palette-row on' : 'palette-row'}
              type="button"
              onMouseEnter={() => setIdx(i)}
              onClick={() => void run(item)}
            >
              <span className="accent">{item.kind}</span> {item.title}
              {item.subtitle ? <span className="muted">  {item.subtitle}</span> : null}
              {item.keys ? <span className="muted">  [{item.keys}]</span> : null}
            </button>
          ))
        )}
      </div>
    </div>
  )
}

function filterPalette(items: PaletteItem[], q: string, repoId: string) {
  const query = q.trim().toLowerCase()
  let list = items
  if (query) {
    list = items.filter((item) => {
      const hay = `${item.title} ${item.subtitle ?? ''} ${item.action ?? ''} ${item.repo_id ?? ''}`.toLowerCase()
      return hay.includes(query) || query.split(/\s+/).every((part) => hay.includes(part))
    })
  } else if (repoId) {
    const here = items.filter((item) => item.repo_id === repoId || !item.repo_id)
    list = here.length ? here : items
  }
  return list.slice(0, 14)
}
