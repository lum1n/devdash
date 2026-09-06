import { Link, createRouter } from '@tanstack/react-router'
import { setupRouterSsrQueryIntegration } from '@tanstack/react-router-ssr-query'
import * as TanStackQuery from '#/integrations/tanstack-query/root-provider'
import { routeTree } from './routeTree.gen'

export function getRouter() {
  const queryContext = TanStackQuery.getContext()

  const router = createRouter({
    routeTree,
    context: {
      ...queryContext,
    },
    scrollRestoration: true,
    defaultPreload: 'intent',
    defaultPreloadStaleTime: 0,
    Wrap: ({ children }: { children: React.ReactNode }) => {
      return (
        <TanStackQuery.Provider queryClient={queryContext.queryClient}>{children}</TanStackQuery.Provider>
      )
    },
    defaultNotFoundComponent: () => {
      return (
        <div className="empty muted">
          no such route. <Link to="/">overview</Link>
        </div>
      )
    },
  })

  setupRouterSsrQueryIntegration({
    router,
    queryClient: queryContext.queryClient,
  })

  return router
}

declare module '@tanstack/react-router' {
  interface Register {
    router: ReturnType<typeof getRouter>
  }
}
