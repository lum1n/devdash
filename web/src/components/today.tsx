import { Link, useNavigate } from '@tanstack/react-router'
import { formatAge, toneClass } from '#/lib/format'
import type { TodayItem } from '#/lib/types'

export function TodayBoard({
  items,
  pending,
  onFocus,
  onOpen,
}: {
  items: TodayItem[]
  pending: boolean
  onFocus: (id: string, action: 'pin' | 'unpin' | 'snooze' | 'archive' | 'unarchive') => void
  onOpen: (id: string, action: 'editor' | 'term') => void
}) {
  const navigate = useNavigate()
  return (
    <section className="section">
      <p className="section-title">
        today
        <span className="muted"> {items.length} · enter jump  e editor  t tmux  p pin  s snooze  x archive</span>
      </p>
      {items.length === 0 ? (
        <p className="empty muted">queue is clear — pin a repo, set a next action, or wait for dirty / behind / agents</p>
      ) : (
        <table className="data">
          <thead>
            <tr>
              <th>name</th>
              <th>why</th>
              <th>last</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => {
              const r = item.repo
              return (
                <tr key={r.id} onClick={() => navigate({ to: '/repos/$repoId', params: { repoId: r.id } })}>
                  <td>
                    {item.pinned ? <span className="accent">* </span> : null}
                    <Link to="/repos/$repoId" params={{ repoId: r.id }} onClick={(e) => e.stopPropagation()}>
                      {r.name}
                    </Link>
                    {item.next ? <div className="today-next">{item.next}</div> : null}
                  </td>
                  <td>
                    {item.why.map((w) => (
                      <span key={w.id + w.label} className={`signal ${toneClass(w.tone)}`}>
                        {w.label}
                        {w.detail ? <span className="muted"> {w.detail}</span> : null}
                      </span>
                    ))}
                  </td>
                  <td className="muted">{formatAge(item.opened_at || r.last_commit)}</td>
                  <td>
                    <span className="today-actions">
                      <button
                        className="tab"
                        type="button"
                        disabled={pending}
                        onClick={(e) => {
                          e.stopPropagation()
                          onOpen(r.id, 'editor')
                        }}
                      >
                        e
                      </button>
                      <button
                        className="tab"
                        type="button"
                        disabled={pending}
                        onClick={(e) => {
                          e.stopPropagation()
                          onOpen(r.id, 'term')
                        }}
                      >
                        t
                      </button>
                      <button
                        className="tab"
                        type="button"
                        disabled={pending}
                        onClick={(e) => {
                          e.stopPropagation()
                          onFocus(r.id, item.pinned ? 'unpin' : 'pin')
                        }}
                      >
                        p
                      </button>
                      <button
                        className="tab"
                        type="button"
                        disabled={pending}
                        onClick={(e) => {
                          e.stopPropagation()
                          onFocus(r.id, 'snooze')
                        }}
                      >
                        s
                      </button>
                      <button
                        className="tab"
                        type="button"
                        disabled={pending}
                        onClick={(e) => {
                          e.stopPropagation()
                          onFocus(r.id, 'archive')
                        }}
                      >
                        x
                      </button>
                    </span>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}
    </section>
  )
}
