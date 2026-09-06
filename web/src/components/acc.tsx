import { formatUSD, sparkline, toneClass } from '#/lib/format'
import type { AccOverviewData, AccProjectData, AccSession, Widget } from '#/lib/types'

export function AccOverview({ widgets }: { widgets: Widget[] | null }) {
  const data = widgetData<AccOverviewData>(widgets, 'acc.overview')
  if (!data) return null
  return (
    <section className="section">
      <p className="section-title">acc</p>
      <div className="acc-grid">
        <div>
          <p className="muted">fleet</p>
          <table className="kv">
            <tbody>
              <tr>
                <td>agents</td>
                <td>{data.fleet.agents}</td>
              </tr>
              <tr>
                <td>attention</td>
                <td className={data.fleet.attention ? 'danger' : ''}>{data.fleet.attention}</td>
              </tr>
              <tr>
                <td>watcher</td>
                <td className={data.fleet.connected ? 'accent' : 'warn'}>
                  {data.fleet.connected ? 'live' : 'off'}
                </td>
              </tr>
              <tr>
                <td>skillcp</td>
                <td className={data.fleet.skillcp_ok ? 'accent' : 'muted'}>
                  {data.fleet.skillcp_ok ? 'ok' : '—'}
                </td>
              </tr>
            </tbody>
          </table>
          {data.fleet.error ? <p className="muted">{data.fleet.error}</p> : null}
        </div>
        <div>
          <p className="muted">plans</p>
          {data.plans.length === 0 ? (
            <p className="empty muted">no meters</p>
          ) : (
            data.plans.map((p) => (
              <div key={p.id} className="meter">
                <span className={toneClass(planTone(p.health, p.used_pct))}>{p.name}</span>
                <div className="meter-track" aria-hidden>
                  <div
                    className={`meter-fill ${toneClass(planTone(p.health, p.used_pct))}`}
                    style={{ width: `${Math.min(100, Math.max(0, p.used_pct))}%` }}
                  />
                </div>
                <span className="muted">
                  {Math.round(p.used_pct)}%{p.label ? ` ${p.label}` : ''}
                </span>
              </div>
            ))
          )}
        </div>
        <div>
          <p className="muted">spend</p>
          <table className="kv">
            <tbody>
              <tr>
                <td>today</td>
                <td>{formatUSD(data.cost.today)}</td>
              </tr>
              <tr>
                <td>week</td>
                <td>{formatUSD(data.cost.week)}</td>
              </tr>
              <tr>
                <td>30d</td>
                <td>{formatUSD(data.cost.month)}</td>
              </tr>
            </tbody>
          </table>
          <p className="spark" aria-label="cost sparkline">
            {sparkline(data.cost.spark)}
          </p>
        </div>
      </div>
    </section>
  )
}

export function AccProject({
  widgets,
  pending,
  onLaunch,
  onResume,
}: {
  widgets?: Widget[]
  pending: boolean
  onLaunch: (harness: string) => void
  onResume: (session: AccSession) => void
}) {
  const data = widgetData<AccProjectData>(widgets, 'acc.project')
  if (!data) return null
  return (
    <section className="section">
      <p className="section-title">
        acc
        <span className="muted"> {formatUSD(data.cost_30d)} 30d</span>
      </p>
      <div className="acc-project">
        <div>
          <p className="muted">fleet on this path</p>
          {data.agents.length === 0 ? (
            <p className="empty muted">no live agents</p>
          ) : (
            <table className="data">
              <tbody>
                {data.agents.map((a) => (
                  <tr key={`${a.session}-${a.kind}`}>
                    <td className="accent">{a.kind}</td>
                    <td className={toneClass(agentTone(a.state))}>{a.state}</td>
                    <td className="muted">{a.session}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
        <div className="acc-actions">
          <div>
            <p className="section-title">launch</p>
            <div className="shortcut-row">
              {data.harnesses.map((h) => (
                <button
                  key={h.id}
                  className="btn shortcut"
                  type="button"
                  disabled={!h.ready || pending}
                  onClick={() => onLaunch(h.id)}
                >
                  <span className="accent">[{h.id.slice(0, 1)}]</span> {h.label}
                </button>
              ))}
            </div>
          </div>
          {data.sessions.length ? (
            <div>
              <p className="section-title">resume</p>
              <div className="shortcut-row">
                {data.sessions.slice(0, 4).map((s) => (
                  <button
                    key={`${s.source}-${s.id}`}
                    className="btn shortcut"
                    type="button"
                    disabled={pending}
                    onClick={() => onResume(s)}
                  >
                    {s.source} {s.title || s.id.slice(0, 8)}
                  </button>
                ))}
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </section>
  )
}

export function widgetData<T>(widgets: Widget[] | null | undefined, kind: string): T | null {
  const w = widgets?.find((item) => item.kind === kind)
  if (!w || !w.data || typeof w.data !== 'object') return null
  return w.data as T
}

function planTone(health: string, pct: number) {
  if (health === 'error' || health === 'auth_required' || pct >= 90) return 'danger'
  if (pct >= 75) return 'warn'
  if (health === 'ok') return 'ok'
  return 'muted'
}

function agentTone(state: string) {
  if (state === 'waiting-permission' || state === 'errored') return 'danger'
  if (state === 'thinking' || state === 'running-tool') return 'warn'
  return 'ok'
}
