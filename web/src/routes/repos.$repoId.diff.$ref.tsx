import { useSuspenseQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { DiffPage } from '#/components/diff'
import { diffQueryOptions } from '#/lib/query-options'
import { cleanRepoId } from '#/lib/repo-id'

export const Route = createFileRoute('/repos/$repoId/diff/$ref')({
  validateSearch: (search: Record<string, unknown>) => ({
    path: typeof search.path === 'string' && search.path ? search.path : undefined,
  }),
  loader: async ({ context, params, location }) => {
    const id = cleanRepoId(params.repoId)
    const path = searchPath(location.search)
    await context.queryClient.ensureQueryData(diffQueryOptions(id, params.ref, path))
  },
  component: DiffRoute,
})

function searchPath(search: unknown) {
  if (search && typeof search === 'object' && 'path' in search && typeof search.path === 'string' && search.path) {
    return search.path
  }
  return undefined
}

function DiffRoute() {
  const { repoId: rawId, ref } = Route.useParams()
  const { path } = Route.useSearch()
  const repoId = cleanRepoId(rawId)
  const { data } = useSuspenseQuery(diffQueryOptions(repoId, ref, path))
  return <DiffPage repoId={repoId} diff={data} />
}
