import { useMutation, useQueryClient, useSuspenseQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { overviewQueryOptions } from '#/lib/query-options'
import { addWorkspace, setFocus } from '#/lib/server-functions'

export const Route = createFileRoute('/settings')({
  loader: async ({ context }) => {
    await context.queryClient.ensureQueryData(overviewQueryOptions())
  },
  component: SettingsPage,
})

function SettingsPage() {
  const queryClient = useQueryClient()
  const { data } = useSuspenseQuery(overviewQueryOptions())
  const focus = useMutation({
    mutationFn: (id: string) => setFocus({ data: { id, action: 'unarchive' } }),
    onSuccess: (ov) => queryClient.setQueryData(['overview'], ov),
  })
  const [wsName, setWsName] = useState('')
  const [wsKind, setWsKind] = useState<'local' | 'ssh'>('local')
  const [wsRoots, setWsRoots] = useState('')
  const [wsHost, setWsHost] = useState('')
  const [wsURL, setWsURL] = useState('')
  const addWs = useMutation({
    mutationFn: () =>
      addWorkspace({
        data: {
          name: wsName.trim(),
          kind: wsKind,
          roots: wsKind === 'local' ? wsRoots.split('\n').map((s) => s.trim()).filter(Boolean) : [],
          host: wsKind === 'ssh' ? wsHost.trim() : undefined,
          url: wsKind === 'ssh' ? wsURL.trim() : undefined,
        },
      }),
    onSuccess: (ov) => {
      queryClient.setQueryData(['overview'], ov)
      setWsName('')
      setWsRoots('')
      setWsHost('')
      setWsURL('')
    },
  })
  return (
    <>
      <header className="topbar">
        <h2 className="page-title">settings</h2>
      </header>
      <div className="content">
        <section className="section">
          <p className="section-title">workspaces</p>
          <table className="data">
            <tbody>
              {(data.workspaces ?? []).map((w) => (
                <tr key={w.id}>
                  <td className={w.active ? 'accent' : ''}>{w.name}</td>
                  <td className="muted">{w.kind}</td>
                  <td className="muted">{w.kind === 'ssh' ? w.host || w.url : (w.roots ?? []).join(' · ')}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <form
            className="next-row"
            onSubmit={(e) => {
              e.preventDefault()
              if (wsName.trim()) addWs.mutate()
            }}
          >
            <input aria-label="workspace name" placeholder="work" value={wsName} onChange={(e) => setWsName(e.target.value)} />
            <select aria-label="kind" value={wsKind} onChange={(e) => setWsKind(e.target.value as 'local' | 'ssh')}>
              <option value="local">local</option>
              <option value="ssh">ssh</option>
            </select>
            {wsKind === 'local' ? (
              <input
                aria-label="roots"
                placeholder="/home/you/work"
                value={wsRoots}
                onChange={(e) => setWsRoots(e.target.value)}
              />
            ) : (
              <>
                <input aria-label="ssh host" placeholder="user@host" value={wsHost} onChange={(e) => setWsHost(e.target.value)} />
                <input
                  aria-label="remote api url"
                  placeholder="127.0.0.1:8790"
                  value={wsURL}
                  onChange={(e) => setWsURL(e.target.value)}
                />
              </>
            )}
            <button className="btn" type="submit" disabled={addWs.isPending || !wsName.trim()}>
              add
            </button>
          </form>
          {addWs.error ? <p className="danger">{addWs.error.message}</p> : null}
          <p className="muted">ssh: remote runs `devdash serve`; url is the local side of ssh -L (or we open the hop)</p>
        </section>
        <section className="section">
          <p className="section-title">roots</p>
          {(data.roots ?? []).length === 0 ? (
            <p className="empty muted">none yet — add one from overview</p>
          ) : (
            <table className="data">
              <tbody>
                {(data.roots ?? []).map((root) => (
                  <tr key={root}>
                    <td>{root}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          <p className="muted">config: ~/.config/devdash/config.yaml</p>
        </section>
        <section className="section">
          <p className="section-title">focus</p>
          <table className="kv">
            <tbody>
              <tr>
                <td>pinned</td>
                <td>{data.focus?.pinned?.length ? data.focus.pinned.join(' · ') : '—'}</td>
              </tr>
              <tr>
                <td>snoozed</td>
                <td>{data.focus?.snoozed?.length ? data.focus.snoozed.join(' · ') : '—'}</td>
              </tr>
              <tr>
                <td>archived</td>
                <td>
                  {data.focus?.archived?.length
                    ? data.focus.archived.map((id) => (
                        <span key={id} className="today-actions">
                          {id}{' '}
                          <button
                            className="tab"
                            type="button"
                            disabled={focus.isPending}
                            onClick={() => focus.mutate(id)}
                          >
                            unarchive
                          </button>
                        </span>
                      ))
                    : '—'}
                </td>
              </tr>
            </tbody>
          </table>
          <p className="muted">archived filter on overview · x toggles archive</p>
        </section>
      </div>
    </>
  )
}
