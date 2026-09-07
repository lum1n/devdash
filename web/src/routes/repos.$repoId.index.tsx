import { useSuspenseQuery } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useEffect } from 'react'
import { ProjectPage } from '#/components/project'
import { RouteError } from '#/components/route-error'
import { projectQueryOptions } from '#/lib/query-options'
import { cleanRepoId } from '#/lib/repo-id'

export const Route = createFileRoute('/repos/$repoId/')({
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(projectQueryOptions(cleanRepoId(params.repoId)))
  },
  errorComponent: RouteError,
  component: RepoPage,
})

function RepoPage() {
  const { repoId: raw } = Route.useParams()
  const repoId = cleanRepoId(raw)
  const navigate = useNavigate()
  const { data } = useSuspenseQuery(projectQueryOptions(repoId))

  useEffect(() => {
    if (raw !== repoId) {
      void navigate({ to: '/repos/$repoId', params: { repoId }, replace: true })
    }
  }, [navigate, raw, repoId])

  return <ProjectPage detail={data} />
}
