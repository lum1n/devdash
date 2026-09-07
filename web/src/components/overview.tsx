import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { AccOverview } from '#/components/acc'
import { LivePlugins } from '#/components/plugins'
import { TodayBoard } from '#/components/today'
import { formatAge } from '#/lib/format'
import { addRoot, openProject, rescan, setFocus } from '#/lib/server-functions'
import type { Annotation, Overview, Repo } from '#/lib/types'

type Filter = 'all' | 'dirty' | 'stale' | 'go' | 'ts' | 'archived'

export function OverviewPage({ overview }: { overview: Overview }) {
  const queryClient = useQueryClient()
  const [filter, setFilter] = useState<Filter>('all')
  const [q, setQ] = useState('')
  const navigate = useNavigate()
  const [rootPath, setRootPath] = useState('')
  const archivedIds = overview.focus?.archived ?? []

  const scan = useMutation({
    mutationFn: () => rescan(),
    onSuccess: (data) => {
      queryClient.setQueryData(['overview'], data)
    },
  })
  const add = useMutation({
    mutationFn: (path: string) => addRoot({ data: { path } }),
    onSuccess: (data) => {
      queryClient.setQueryData(['overview'], data)
      setRootPath('')
    },
  })
  const focusMut = useMutation({
    mutationFn: (input: { id: string; action: 'pin' | 'unpin' | 'snooze' | 'archive' | 'unarchive' }) =>
      setFocus({ data: input }),
    onSuccess: (data) => queryClient.setQueryData(['overview'], data),
  })
  const openMut = useMutation({
    mutationFn: (input: { id: string; action: 'editor' | 'term' }) => openProject({ data: input }),
  })

  const repos = useMemo(() => {
    const now = Date.now()
    const archived = new Set(archivedIds)
    return (overview.repos ?? []).filter((r) => {
      const isArchived = archived.has(r.id)
      if (filter === 'archived') return isArchived
      if (isArchived) return false
      if (q && !r.name.toLowerCase().includes(q.toLowerCase())) return false
      if (filter === 'dirty') return r.dirty
      if (filter === 'stale') {
        const t = r.last_commit ? Date.parse(r.last_commit) : 0
        return !t || now - t > 14 * 24 * 60 * 60 * 1000
      }
      if (filter === 'go' || filter === 'ts') return r.stack === filter
      return true
    })
  }, [overview.repos, filter, q, archivedIds])

  return (
    <>
      <header className="topbar">
        <h2 className="page-title">overview</h2>
        <div className="flex flex-wrap gap-2">
          <input
            aria-label="filter repos"
            placeholder="/ filter   : palette"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          <button className="btn" type="button" onClick={() => scan.mutate()} disabled={scan.isPending}>
            {scan.isPending ? 'scanning' : 'rescan'}
          </button>
        </div>
      </header>
      <div className="content">
        <section className="section now">
          <p className="section-title">now</p>
          <div className="now-row" role="group" aria-label="now">
            <NowMetric label="repos" value={overview.counts.repos} on={filter === 'all'} onClick={() => setFilter('all')} />
            <NowMetric
              label="dirty"
              value={overview.counts.dirty}
              tone={overview.counts.dirty ? 'warn' : ''}
              on={filter === 'dirty'}
              onClick={() => setFilter('dirty')}
            />
            <NowMetric label="behind" value={overview.counts.behind} tone={overview.counts.behind ? 'danger' : ''} />
            <NowMetric label="ahead" value={overview.counts.ahead} />
            <NowMetric label="attention" value={overview.counts.attention} tone="accent" />
            <NowMetric label="today" value={overview.counts.today} tone="accent" />
            <NowMetric
              label="archived"
              value={overview.counts.archived}
              on={filter === 'archived'}
              onClick={() => setFilter('archived')}
            />
          </div>
        </section>

        <TodayBoard
          items={overview.today ?? []}
          pending={focusMut.isPending || openMut.isPending}
          onFocus={(id, action) => focusMut.mutate({ id, action })}
          onOpen={(id, action) => openMut.mutate({ id, action })}
        />

        <AccOverview widgets={overview.widgets} />
        <LivePlugins widgets={overview.widgets} />

        <section className="section">
          <p className="section-title">add root</p>
          <form
            className="flex flex-wrap items-center gap-2"
            onSubmit={(e) => {
              e.preventDefault()
              if (rootPath.trim()) add.mutate(rootPath.trim())
            }}
          >
            <input
              aria-label="root path"
              placeholder="/home/you/repos"
              value={rootPath}
              onChange={(e) => setRootPath(e.target.value)}
              className="min-w-0 w-full sm:min-w-72 sm:w-auto"
            />
            <button className="btn primary" type="submit" disabled={add.isPending}>
              add
            </button>
            {add.error ? <span className="danger">{add.error.message}</span> : null}
          </form>
        </section>

        <section className="section">
          <div className="mb-2 flex flex-wrap gap-3">
            {(['all', 'dirty', 'stale', 'go', 'ts', 'archived'] as const).map((f) => (
              <button key={f} className={filter === f ? 'tab active' : 'tab'} type="button" onClick={() => setFilter(f)}>
                {f}
              </button>
            ))}
          </div>
          {repos.length === 0 ? (
            <p className="empty muted">
              {filter === 'archived' ? 'nothing archived' : 'no repos — add a root that contains git checkouts'}
            </p>
          ) : (
            <table className="data">
              <thead>
                <tr>
                  <th>name</th>
                  <th>branch</th>
                  <th>stack</th>
                  <th>status</th>
                  <th>last</th>
                  {filter === 'archived' ? <th></th> : null}
                </tr>
              </thead>
              <tbody>
                {repos.map((r) => (
                  <tr key={r.path} onClick={() => navigate({ to: '/repos/$repoId', params: { repoId: r.id } })}>
                    <td>
                      <Link to="/repos/$repoId" params={{ repoId: r.id }} onClick={(e) => e.stopPropagation()}>
                        {r.name}
                      </Link>
                    </td>
                    <td className="muted">{r.branch || '—'}</td>
                    <td className="muted">{r.stack || '—'}</td>
                    <td>
                      <Status repo={r} notes={notesFor(overview.annotations, r.id)} />
                    </td>
                    <td className="muted">{formatAge(r.last_commit)}</td>
                    {filter === 'archived' ? (
                      <td>
                        <button
                          className="tab"
                          type="button"
                          disabled={focusMut.isPending}
                          onClick={(e) => {
                            e.stopPropagation()
                            focusMut.mutate({ id: r.id, action: 'unarchive' })
                          }}
                        >
                          unarchive
                        </button>
                      </td>
                    ) : null}
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>

      </div>
    </>
  )
}

function NowMetric({
  label,
  value,
  tone,
  on,
  onClick,
}: {
  label: string
  value: number
  tone?: string
  on?: boolean
  onClick?: () => void
}) {
  const className = ['now-metric', on ? 'on' : '', onClick ? 'hit' : ''].filter(Boolean).join(' ')
  const inner = (
    <>
      <span className="now-label">{label}</span>
      <span className={`now-value ${tone ?? ''}`}>{value}</span>
    </>
  )
  if (onClick) {
    return (
      <button className={className} type="button" onClick={onClick}>
        {inner}
      </button>
    )
  }
  return <div className={className}>{inner}</div>
}

function notesFor(notes: Annotation[] | null, id: string) {
  return (notes ?? []).filter((n) => n.repo_id === id)
}

function Status({ repo, notes }: { repo: Repo; notes: Annotation[] }) {
  const clean = !repo.dirty && !repo.behind && !repo.ahead
  return (
    <span>
      {clean ? <span className="accent">clean </span> : null}
      {repo.dirty ? <span className="warn">+{repo.changed} </span> : null}
      {repo.behind ? <span className="danger">↓{repo.behind} </span> : null}
      {repo.ahead ? <span className="warn">↑{repo.ahead} </span> : null}
      {notes.map((n) => (
        <span key={`${n.plugin}-${n.kind}`} className={n.tone === 'danger' ? 'danger' : n.tone === 'warn' ? 'warn' : 'accent'}>
          {n.label}
        </span>
      ))}
    </span>
  )
}

