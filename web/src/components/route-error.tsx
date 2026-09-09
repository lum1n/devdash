import type { ErrorComponentProps } from '@tanstack/react-router'

export function RouteError({ error }: ErrorComponentProps) {
  const message = error instanceof Error ? error.message : 'unknown error'
  return (
    <div className="content">
      <p className="section-title">error</p>
      <p className="empty danger">{message}</p>
      <p className="muted">switch workspace in the sidebar, or check compose logs</p>
      <p className="muted">
        api: <code>docker compose logs api</code>
      </p>
    </div>
  )
}
