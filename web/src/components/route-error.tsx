import type { ErrorComponentProps } from '@tanstack/react-router'

export function RouteError({ error }: ErrorComponentProps) {
  const message = error instanceof Error ? error.message : 'unknown error'
  return (
    <div className="content">
      <p className="section-title">error</p>
      <p className="empty danger">{message}</p>
      <p className="muted">
        api: <code>go run ./cmd/devdash serve</code>
      </p>
    </div>
  )
}
