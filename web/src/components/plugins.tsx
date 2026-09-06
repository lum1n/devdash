import { widgetData } from '#/components/acc'
import { toneClass } from '#/lib/format'
import type { GhOverviewData, GhProjectData, PortsOverviewData, PortsProjectData, TmuxOverviewData, TmuxProjectData, Widget } from '#/lib/types'

export function LivePlugins({ widgets }: { widgets: Widget[] | null }) {
  const tmux = widgetData<TmuxOverviewData>(widgets, 'tmux.overview')
  const ports = widgetData<PortsOverviewData>(widgets, 'ports.overview')
  const gh = widgetData<GhOverviewData>(widgets, 'gh.overview')
  if (!tmux && !ports && !gh) return null
  return (
    <section className="section">
      <p className="section-title">live</p>
      <div className="acc-grid live-plugins">
        {tmux ? (
          <div>
            <p className="muted">tmux</p>
            <p className={tmux.ready ? 'accent' : 'warn'}>{tmux.sessions} sessions</p>
            {tmux.error ? <p className="muted">{tmux.error}</p> : null}
          </div>
        ) : null}
        {ports ? (
          <div>
            <p className="muted">ports</p>
            <p className={ports.listening ? 'accent' : 'muted'}>{ports.listening} listening</p>
            {ports.error ? <p className="muted">{ports.error}</p> : null}
          </div>
        ) : null}
        {gh ? (
          <div>
            <p className="muted">gh</p>
            <p>
              <span className={gh.open ? 'accent' : 'muted'}>{gh.open} open</span>
              {gh.review ? <span className="danger"> · {gh.review} review</span> : null}
            </p>
            {gh.error ? <p className="muted">{gh.error}</p> : null}
          </div>
        ) : null}
      </div>
    </section>
  )
}

export function TmuxProject({
  widgets,
  pending,
  onAttach,
}: {
  widgets?: Widget[]
  pending: boolean
  onAttach: (session?: string) => void
}) {
  const data = widgetData<TmuxProjectData>(widgets, 'tmux.project')
  if (!data) return null
  return (
    <section className="section">
      <p className="section-title">tmux</p>
      {data.sessions.length === 0 ? (
        <p className="empty muted">{data.ready ? 'no session' : 'tmux off'}</p>
      ) : (
        <table className="data">
          <tbody>
            {data.sessions.map((s) => (
              <tr key={s.name}>
                <td className="accent">{s.name}</td>
                <td className="muted">{s.windows.map((w) => w.name || String(w.index)).join(' ')}</td>
                <td>
                  <button className="btn shortcut" type="button" disabled={pending} onClick={() => onAttach(s.name)}>
                    attach
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  )
}

export function PortsProject({
  widgets,
  pending,
  onOpen,
}: {
  widgets?: Widget[]
  pending: boolean
  onOpen: (url?: string) => void
}) {
  const data = widgetData<PortsProjectData>(widgets, 'ports.project')
  if (!data) return null
  return (
    <section className="section">
      <p className="section-title">ports</p>
      {data.listeners.length === 0 ? (
        <p className="empty muted">none listening</p>
      ) : (
        <table className="data">
          <tbody>
            {data.listeners.map((l) => (
              <tr key={`${l.pid}-${l.port}`}>
                <td className="accent">
                  {l.label} :{l.port}
                </td>
                <td className="muted">{l.command}</td>
                <td>
                  <button className="btn shortcut" type="button" disabled={pending} onClick={() => onOpen(l.url)}>
                    open
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  )
}

export function GhProject({
  widgets,
  pending,
  onOpen,
}: {
  widgets?: Widget[]
  pending: boolean
  onOpen: (url?: string) => void
}) {
  const data = widgetData<GhProjectData>(widgets, 'gh.project')
  if (!data) return null
  return (
    <section className="section">
      <p className="section-title">
        gh
        {data.slug ? <span className="muted"> {data.slug}</span> : null}
      </p>
      {data.pulls.length === 0 ? (
        <p className="empty muted">no open prs</p>
      ) : (
        <table className="data">
          <tbody>
            {data.pulls.map((pr) => (
              <tr key={pr.number}>
                <td className="accent">#{pr.number}</td>
                <td className="lead">{pr.title}</td>
                <td className={toneClass(pr.review || pr.checks === 'fail' ? 'danger' : 'ok')}>
                  {pr.review ? 'review' : pr.checks === 'fail' ? 'ci' : pr.draft ? 'draft' : 'open'}
                </td>
                <td>
                  <button className="btn shortcut" type="button" disabled={pending} onClick={() => onOpen(pr.url)}>
                    open
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  )
}
