import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { workspacesQueryOptions } from '#/lib/query-options'
import { selectWorkspace } from '#/lib/server-functions'
import type { Workspace } from '#/lib/types'

export function WorkspacePicker({ spaces }: { spaces?: Workspace[] | null }) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const remote = useQuery(workspacesQueryOptions())
  const list = remote.data?.workspaces ?? spaces ?? []
  const active = list.find((w) => w.active) ?? list[0]
  const select = useMutation({
    mutationFn: (id: string) => selectWorkspace({ data: { id } }),
    onSuccess: (ov) => {
      queryClient.setQueryData(['overview'], ov)
      queryClient.setQueryData(['workspaces'], {
        active: ov.workspace?.id,
        workspaces: ov.workspaces ?? [],
      })
      queryClient.removeQueries({ queryKey: ['project'] })
      void navigate({ to: '/' })
    },
  })
  if (!list.length || !active) return null

  return (
    <label className="ws-picker">
      <span className="ws-picker-label">ws</span>
      <select
        aria-label="workspace"
        value={active.id}
        disabled={select.isPending || list.length === 1}
        title={select.error instanceof Error ? select.error.message : undefined}
        onChange={(e) => {
          const id = e.target.value
          if (id && id !== active.id) select.mutate(id)
        }}
      >
        {list.map((w) => (
          <option key={w.id} value={w.id}>
            {optionLabel(w)}
          </option>
        ))}
      </select>
    </label>
  )
}

function optionLabel(w: Workspace) {
  const bits = [w.name]
  if (w.kind === 'ssh') bits.push('ssh')
  if (!w.ready) bits.push('…')
  return bits.join(' · ')
}
