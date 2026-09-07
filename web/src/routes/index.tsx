import { useSuspenseQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { OverviewPage } from '#/components/overview'
import { RouteError } from '#/components/route-error'
import { overviewQueryOptions } from '#/lib/query-options'

export const Route = createFileRoute('/')({
  loader: async ({ context }) => {
    await context.queryClient.ensureQueryData(overviewQueryOptions())
  },
  errorComponent: RouteError,
  component: IndexPage,
})

function IndexPage() {
  const { data } = useSuspenseQuery(overviewQueryOptions())
  return <OverviewPage overview={data} />
}
