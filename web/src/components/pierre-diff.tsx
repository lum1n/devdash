import { parsePatchFiles, type CodeViewItem } from '@pierre/diffs'
import { CodeView, type CodeViewReactOptions } from '@pierre/diffs/react'
import { useMemo } from 'react'

export type DiffStyle = 'unified' | 'split'

const codeViewStyle = { height: '100%', minHeight: '24rem', overflow: 'auto' } as const

export function PierreDiff({
  patch,
  cacheKey,
  diffStyle,
}: {
  patch: string
  cacheKey: string
  diffStyle: DiffStyle
}) {
  const items = useMemo(() => itemsFromPatch(patch, cacheKey), [patch, cacheKey])
  const options = useMemo<CodeViewReactOptions<undefined, undefined>>(
    () => ({
      theme: 'pierre-dark',
      stickyHeaders: true,
      overflow: 'scroll',
      diffStyle,
    }),
    [diffStyle],
  )
  if (items.length === 0) {
    return <p className="empty muted">could not parse patch</p>
  }
  return (
    <CodeView
      key={diffStyle}
      items={items}
      disableWorkerPool
      options={options}
      className="diff-view"
      style={codeViewStyle}
    />
  )
}

function itemsFromPatch(patch: string, cacheKey: string): CodeViewItem<undefined>[] {
  try {
    return parsePatchFiles(patch, cacheKey).flatMap((p, i) =>
      p.files.map((fileDiff, j) => ({
        id: `${i}:${j}:${fileDiff.name}`,
        type: 'diff' as const,
        fileDiff,
      })),
    )
  } catch {
    return []
  }
}
