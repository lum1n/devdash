import {
  HeadContent,
  Link,
  Outlet,
  Scripts,
  createRootRouteWithContext,
  useRouterState,
} from '@tanstack/react-router'
import type { QueryClient } from '@tanstack/react-query'
import { CommandPalette } from '#/components/palette'
import appCss from '../styles.css?url'

interface RouterContext {
  queryClient: QueryClient
}

export const Route = createRootRouteWithContext<RouterContext>()({
  head: () => ({
    meta: [
      { charSet: 'utf-8' },
      { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      { title: 'devdash' },
      { name: 'description', content: 'Local project dashboard' },
    ],
    links: [{ rel: 'stylesheet', href: appCss }],
  }),
  errorComponent: RootError,
  shellComponent: RootDocument,
})

function RootError({ error }: { error: unknown }) {
  const message = error instanceof Error ? error.message : 'unknown error'
  return (
    <div className="content">
      <p className="section-title">startup</p>
      <p className="empty danger">{message}</p>
      <p className="muted">
        start the go api: <code>go run ./cmd/devdash serve</code>
      </p>
    </div>
  )
}

function RootDocument() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  return (
    <html lang="en">
      <head>
        <HeadContent />
      </head>
      <body>
        <div className="shell">
          <aside className="sidebar">
            <div className="brand">devdash</div>
            <nav className="nav">
              <Link className={navClass(pathname, '/')} to="/">
                overview
              </Link>
              <Link className={navClass(pathname, '/settings')} to="/settings">
                settings
              </Link>
              {pathname.startsWith('/repos/') ? (
                pathname.includes('/notes/') ? (
                  <span className="nav-link active">note</span>
                ) : pathname.includes('/diff/') ? (
                  <span className="nav-link active">diff</span>
                ) : (
                  <span className="nav-link active">project</span>
                )
              ) : null}
            </nav>
            <div className="sidebar-foot">loopback</div>
          </aside>
          <div className="main">
            <Outlet />
          </div>
        </div>
        <CommandPalette />
        <Scripts />
      </body>
    </html>
  )
}

function navClass(pathname: string, href: string) {
  const active = href === '/' ? pathname === '/' : pathname.startsWith(href)
  return active ? 'nav-link active' : 'nav-link'
}
