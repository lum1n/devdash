import { Link } from '@tanstack/react-router'
import { useEffect, useState, type ComponentType } from 'react'
import type { DiffStyle } from '#/components/pierre-diff'
import { PageTrail, useRepoName } from '#/components/trail'
import type { RepoDiff } from '#/lib/types'

type DiffView = ComponentType<{ patch: string; cacheKey: string; diffStyle: DiffStyle }>

export function DiffPage({ repoId, diff }: { repoId: string; diff: RepoDiff }) {
  const [diffStyle, setDiffStyle] = useState<DiffStyle>('unified')
  const repoName = useRepoName(repoId)
  const fileNote = diff.files.length === 1 ? '1 file' : `${diff.files.length} files`

  return (
    <>
      <header className="topbar">
        <PageTrail
          crumbs={[
            { label: 'overview', to: '/' },
            { label: repoName, to: '/repos/$repoId', repoId },
            { label: diff.title || diff.ref },
          ]}
        />
        <div className="flex flex-wrap items-center gap-2">
          {diff.truncated ? <span className="warn">truncated</span> : null}
          <div role="group" aria-label="diff layout">
            <button
              className={diffStyle === 'unified' ? 'tab active' : 'tab'}
              type="button"
              onClick={() => setDiffStyle('unified')}
            >
              unified
            </button>{' '}
            <button
              className={diffStyle === 'split' ? 'tab active' : 'tab'}
              type="button"
              onClick={() => setDiffStyle('split')}
            >
              split
            </button>
          </div>
          {diff.ref !== 'worktree' ? (
            <button
              className="btn"
              type="button"
              onClick={() => {
                void navigator.clipboard.writeText(diff.ref)
              }}
            >
              copy
            </button>
          ) : null}
          <Link className="btn" to="/repos/$repoId" params={{ repoId }}>
            back
          </Link>
        </div>
      </header>
      <div className="content diff-screen">
        {diff.empty ? (
          <p className="empty muted">no changes</p>
        ) : (
          <ClientDiff
            patch={diff.patch}
            cacheKey={`${repoId}:${diff.ref}:${diff.path ?? ''}`}
            diffStyle={diffStyle}
          />
        )}
        <p className="muted">
          {fileNote} · {diff.path || diff.ref}
        </p>
      </div>
    </>
  )
}

function ClientDiff({
  patch,
  cacheKey,
  diffStyle,
}: {
  patch: string
  cacheKey: string
  diffStyle: DiffStyle
}) {
  const [View, setView] = useState<DiffView | null>(null)

  useEffect(() => {
    let alive = true
    void import('#/components/pierre-diff').then((m) => {
      if (alive) setView(() => m.PierreDiff)
    })
    return () => {
      alive = false
    }
  }, [])

  if (!View) return <p className="muted">loading</p>
  return <View patch={patch} cacheKey={cacheKey} diffStyle={diffStyle} />
}
