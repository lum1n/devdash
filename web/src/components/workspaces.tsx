import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { selectWorkspace } from '#/lib/server-functions'
import type { Workspace } from '#/lib/types'

export function WorkspacePicker({ spaces }: { spaces?: Workspace[] | null }) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const list = spaces ?? []
  const active = list.find((w) => w.active) ?? list[0]
  const select = useMutation({
    mutationFn: (id: string) => selectWorkspace({ data: { id } }),
    onSuccess: (ov) => {
      queryClient.setQueryData(['overview'], ov)
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
