import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useEffect, useMemo, useState } from 'react'
import { AccProject } from '#/components/acc'
import { GhProject, PortsProject, TmuxProject } from '#/components/plugins'
import { ProjectNotes } from '#/components/notes'
import { PageTrail } from '#/components/trail'
import { formatAge, toneClass } from '#/lib/format'
import { statusPath } from '#/lib/repo-id'
import { openProject, runPlugin, setFocus, setNext } from '#/lib/server-functions'
import type { AccSession, ProjectDetail, Shortcut } from '#/lib/types'

const weekdayLabels = ['su', 'mo', 'tu', 'we', 'th', 'fr', 'sa']

export function ProjectPage({ detail }: { detail: ProjectDetail }) {
  const queryClient = useQueryClient()
  const repo = detail.repo
  const [flash, setFlash] = useState('')
  const [week, setWeek] = useState<number | null>(null)
  const [day, setDay] = useState<number | null>(null)
  const [copied, setCopied] = useState('')
  const [nextText, setNextText] = useState(detail.next ?? '')

  useEffect(() => {
    setNextText(detail.next ?? '')
  }, [detail.next])

  const focus = useMutation({
    mutationFn: (action: string) => setFocus({ data: { id: repo.id, action } }),
    onSuccess: (ov, action) => {
      queryClient.setQueryData(['overview'], ov)
      queryClient.setQueryData(['project', repo.id], {
        ...detail,
        archived: action === 'archive' ? true : action === 'unarchive' ? false : detail.archived,
      })
      setFlash(action)
    },
    onError: (err) => setFlash(err.message),
  })

  const acc = useMutation({
    mutationFn: (input: { action: string; harness?: string; session?: string }) =>
      runPlugin({ data: { plugin: 'acc', repo: repo.id, ...input } }),
    onSuccess: (res) => setFlash(res.detail || res.action),
    onError: (err) => setFlash(err.message),
  })
  const plug = useMutation({
    mutationFn: (input: { plugin: string; action: string; target?: string }) =>
      runPlugin({ data: { plugin: input.plugin, repo: repo.id, action: input.action, target: input.target } }),
    onSuccess: (res) => setFlash(res.detail || res.action),
    onError: (err) => setFlash(err.message),
  })

  const next = useMutation({
    mutationFn: (text: string) => setNext({ data: { id: repo.id, text } }),
    onSuccess: (ov) => {
      queryClient.setQueryData(['overview'], ov)
      queryClient.setQueryData(['project', repo.id], { ...detail, next: nextText.trim() })
      setFlash(nextText.trim() ? 'next saved' : 'next cleared')
    },
    onError: (err) => setFlash(err.message),
  })

  const open = useMutation({
    mutationFn: (action: string) => openProject({ data: { id: repo.id, action } }),
    onSuccess: async (res) => {
      if (res.action === 'copy' && res.detail) {
        await navigator.clipboard.writeText(res.detail)
        setCopied(res.detail)
        setFlash('copied path')
        return
      }
      setFlash(res.detail || res.action)
    },
    onError: (err) => setFlash(err.message),
  })

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.metaKey || e.ctrlKey || e.altKey) return
      const tag = (e.target as HTMLElement | null)?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA') return
      if (e.key === 'a') {
        e.preventDefault()
        acc.mutate({ action: 'launch', harness: 'claude' })
        return
      }
      if (e.key === 'p') {
        e.preventDefault()
        focus.mutate('pin')
        return
      }
      if (e.key === 's') {
        e.preventDefault()
        focus.mutate('snooze')
        return
      }
      if (e.key === 'x') {
        e.preventDefault()
        focus.mutate(detail.archived ? 'unarchive' : 'archive')
        return
      }
      const hit = detail.shortcuts.find((s) => s.key === e.key && s.enabled)
      if (hit) {
        e.preventDefault()
        open.mutate(hit.id)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [detail.archived, detail.shortcuts, open, acc, focus])

  const maxWeek = Math.max(1, ...detail.activity.map((w) => w.count))
  const maxDay = Math.max(1, ...detail.weekdays)
  const selectedWeek = week != null ? detail.activity[week] : null

  const commits = useMemo(() => {
    let list = detail.commits
    if (week != null && selectedWeek) {
      const start = Date.parse(selectedWeek.start)
      const end = start + 7 * 24 * 60 * 60 * 1000
      list = list.filter((c) => {
        const t = Date.parse(c.when)
        return t >= start && t < end
      })
    }
    if (day != null) {
      list = list.filter((c) => new Date(c.when).getDay() === day)
    }
    return list
  }, [detail.commits, selectedWeek, week, day])

  return (
    <>
      <header className="topbar">
        <PageTrail crumbs={[{ label: 'overview', to: '/' }, { label: repo.name }]} />
        <div className="flex flex-wrap items-center gap-2">
          {flash ? <span className="accent">{flash}</span> : null}
          <Link className="btn" to="/">
            back
          </Link>
        </div>
      </header>
      <div className="content project-screen">
        {detail.summary ? <p className="project-lede">{detail.summary}</p> : null}

        <section className="section">
          <p className="section-title">meta</p>
          <table className="kv">
            <tbody>
              <tr>
                <td>path</td>
                <td>{repo.path}</td>
              </tr>
              <tr>
                <td>branch</td>
                <td>{repo.branch || '—'}</td>
              </tr>
              <tr>
                <td>stack</td>
                <td>{repo.stack || '—'}</td>
              </tr>
              <tr>
                <td>remote</td>
                <td>
                  {detail.remote_url ? (
                    <a href={detail.remote_url} target="_blank" rel="noreferrer">
                      {detail.remote_url}
                    </a>
                  ) : (
                    repo.remote || '—'
                  )}
                </td>
              </tr>
              <tr>
                <td>authors 30d</td>
                <td>
                  {detail.authors.length
                    ? detail.authors.map((a) => `${a.name} ${a.count}`).join(' · ')
                    : '—'}
                </td>
              </tr>
            </tbody>
          </table>
        </section>

        <section className="section">
          <p className="section-title">next</p>
          <form
            className="next-row"
            onSubmit={(e) => {
              e.preventDefault()
              next.mutate(nextText)
            }}
          >
            <input
              aria-label="next action"
              placeholder="one line — shipping iOS, blocked on API"
              value={nextText}
              maxLength={160}
              onChange={(e) => setNextText(e.target.value)}
            />
            <button className="btn" type="submit" disabled={next.isPending}>
              save
            </button>
          </form>
        </section>

        <ProjectNotes repoId={repo.id} notes={detail.notes ?? []} />

        <AccProject
          widgets={detail.plugins}
          pending={acc.isPending}
          onLaunch={(harness) => acc.mutate({ action: 'launch', harness })}
          onResume={(session: AccSession) =>
            acc.mutate({ action: 'resume', harness: session.source, session: session.id })
          }
        />
        <TmuxProject
          widgets={detail.plugins}
          pending={plug.isPending}
          onAttach={(session) => plug.mutate({ plugin: 'tmux', action: 'attach', target: session })}
        />
        <PortsProject
          widgets={detail.plugins}
          pending={plug.isPending}
          onOpen={(url) => plug.mutate({ plugin: 'ports', action: 'open', target: url })}
        />
        <GhProject
          widgets={detail.plugins}
          pending={plug.isPending}
          onOpen={(url) => plug.mutate({ plugin: 'gh', action: 'open', target: url })}
        />

        <section className="section">
          <p className="section-title">shortcuts</p>
          <div className="shortcut-row">
            {detail.shortcuts.map((s) => (
              <ShortcutButton
                key={s.id}
                shortcut={s}
                pending={open.isPending}
                onRun={() => open.mutate(s.id)}
              />
            ))}
            <button className="btn shortcut" type="button" disabled={focus.isPending} onClick={() => focus.mutate('pin')}>
              <span className="accent">[p]</span> pin
            </button>
            <button className="btn shortcut" type="button" disabled={focus.isPending} onClick={() => focus.mutate('snooze')}>
              <span className="accent">[s]</span> snooze
            </button>
            <button
              className="btn shortcut"
              type="button"
              disabled={focus.isPending}
              onClick={() => focus.mutate(detail.archived ? 'unarchive' : 'archive')}
            >
              <span className="accent">[x]</span> {detail.archived ? 'unarchive' : 'archive'}
            </button>
          </div>
          <p className="muted">
            <span className="keys-hint">
              keys {detail.shortcuts.map((s) => s.key).join(' ')} a p s x · : palette
            </span>
            {copied ? <span>{copied}</span> : null}
          </p>
        </section>

        <section className="section">
          <p className="section-title">signals</p>
          <div className="signal-row">
            {detail.signals.map((s) => (
              <span key={s.id} className={`signal ${toneClass(s.tone)}`}>
                {s.label}
                {s.detail ? <span className="muted"> {s.detail}</span> : null}
              </span>
            ))}
          </div>
        </section>

        <section className="section">
          <p className="section-title">
            activity 16w
            <span className="muted"> {detail.commit_n_16w} commits</span>
            {selectedWeek ? (
              <span className="accent">
                {' '}
                · week of {selectedWeek.start.slice(0, 10)} · {selectedWeek.count}
              </span>
            ) : null}
          </p>
          <div className="chart-weeks" role="img" aria-label="commits per week">
            {detail.activity.map((w, i) => (
              <button
                key={w.start}
                type="button"
                className={week === i ? 'week-bar on' : 'week-bar'}
                style={{ height: `${8 + (w.count / maxWeek) * 56}px` }}
                title={`${w.start.slice(0, 10)} · ${w.count}`}
                onClick={() => setWeek(week === i ? null : i)}
              />
            ))}
          </div>
        </section>

        <section className="section">
          <p className="section-title">weekday</p>
          <div className="chart-days">
            {detail.weekdays.map((n, i) => (
              <button
                key={weekdayLabels[i]}
                type="button"
                className={day === i ? 'day-col on' : 'day-col'}
                onClick={() => setDay(day === i ? null : i)}
              >
                <span className="day-bar" style={{ height: `${6 + (n / maxDay) * 40}px` }} />
                <span className="muted">{weekdayLabels[i]}</span>
                <span>{n}</span>
              </button>
            ))}
          </div>
        </section>

        <div className="project-split">
          <section className="section">
            <p className="section-title">working tree {repo.changed ? `+${repo.changed}` : ''}</p>
            {detail.files.length === 0 ? (
              <p className="empty muted">clean</p>
            ) : (
              <table className="data">
                <tbody>
                  {detail.files.map((f) => (
                    <tr key={f.path}>
                      <td className="warn">{f.status}</td>
                      <td>
                        <Link
                          to="/repos/$repoId/diff/$ref"
                          params={{ repoId: repo.id, ref: 'worktree' }}
                          search={{ path: statusPath(f.path) }}
                        >
                          {f.path}
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </section>

          <section className="section">
            <p className="section-title">commits</p>
            {commits.length === 0 ? (
              <p className="empty muted">none in this slice</p>
            ) : (
              <table className="data">
                <tbody>
                  {commits.map((c) => (
                    <tr key={c.hash}>
                      <td className="muted">{formatAge(c.when)}</td>
                      <td>
                        <button
                          className="hash-copy"
                          type="button"
                          onClick={async () => {
                            await navigator.clipboard.writeText(c.hash)
                            setFlash(`copied ${c.hash}`)
                          }}
                        >
                          {c.hash}
                        </button>
                      </td>
                      <td className="lead">
                        <Link
                          to="/repos/$repoId/diff/$ref"
                          params={{ repoId: repo.id, ref: c.hash }}
                          search={{ path: undefined }}
                        >
                          {c.subject}
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </section>
        </div>
      </div>
    </>
  )
}

function ShortcutButton({
  shortcut,
  pending,
  onRun,
}: {
  shortcut: Shortcut
  pending: boolean
  onRun: () => void
}) {
  return (
    <button className="btn shortcut" type="button" disabled={!shortcut.enabled || pending} onClick={onRun}>
      <span className="accent">[{shortcut.key}]</span> {shortcut.label}
    </button>
  )
}
