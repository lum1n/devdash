import {
  HeadContent,
  Link,
  Outlet,
  Scripts,
  createRootRouteWithContext,
  useRouterState,
} from '@tanstack/react-router'
import type { QueryClient } from '@tanstack/react-query'
import { useQuery } from '@tanstack/react-query'
import { CommandPalette } from '#/components/palette'
import { RouteError } from '#/components/route-error'
import { WorkspacePicker } from '#/components/workspaces'
import { overviewQueryOptions, workspacesQueryOptions } from '#/lib/query-options'
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
  errorComponent: RouteError,
  shellComponent: RootDocument,
})

function RootDocument() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const { data } = useQuery(overviewQueryOptions())
  const workspaces = useQuery(workspacesQueryOptions())

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
            </nav>
            <WorkspacePicker spaces={workspaces.data?.workspaces ?? data?.workspaces} />
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
