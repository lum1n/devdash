import { useMutation, useQueryClient, useSuspenseQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { overviewQueryOptions } from '#/lib/query-options'
import { setFocus } from '#/lib/server-functions'

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
  return (
    <>
      <header className="topbar">
        <h2 className="page-title">settings</h2>
      </header>
      <div className="content">
        <section className="section">
          <p className="section-title">roots</p>
          {data.roots.length === 0 ? (
            <p className="empty muted">none yet — add one from overview</p>
          ) : (
            <table className="data">
              <tbody>
                {data.roots.map((root) => (
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
