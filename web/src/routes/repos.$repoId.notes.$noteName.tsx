import { useSuspenseQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { NoteEditor } from '#/components/notes'
import { RouteError } from '#/components/route-error'
import { noteQueryOptions } from '#/lib/query-options'
import { cleanRepoId, noteFileName } from '#/lib/repo-id'

export const Route = createFileRoute('/repos/$repoId/notes/$noteName')({
  loader: async ({ context, params }) => {
    const id = cleanRepoId(params.repoId)
    const name = noteFileName(params.noteName)
    await context.queryClient.ensureQueryData(noteQueryOptions(id, name))
  },
  errorComponent: RouteError,
  component: NoteRoute,
})

function NoteRoute() {
  const { repoId: rawId, noteName: rawName } = Route.useParams()
  const repoId = cleanRepoId(rawId)
  const noteName = noteFileName(rawName)
  const { data } = useSuspenseQuery(noteQueryOptions(repoId, noteName))
  return <NoteEditor repoId={repoId} note={data} />
}
