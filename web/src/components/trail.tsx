import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { overviewQueryOptions } from '#/lib/query-options'

export function PageTrail({ crumbs }: { crumbs: Crumb[] }) {
  return (
    <h2 className="page-title">
      {crumbs.map((c, i) => (
        <span key={`${c.label}-${i}`}>
          {i > 0 ? <span className="crumb-sep">/</span> : null}
          {c.to && i < crumbs.length - 1 ? <CrumbLink crumb={c} /> : c.label}
        </span>
      ))}
    </h2>
  )
}

export function useRepoName(repoId: string) {
  const { data } = useQuery(overviewQueryOptions())
  return data?.repos.find((r) => r.id === repoId)?.name ?? repoId
}

type Crumb = {
  label: string
  to?: '/' | '/repos/$repoId'
  repoId?: string
}

function CrumbLink({ crumb }: { crumb: Crumb }) {
  if (crumb.to === '/repos/$repoId' && crumb.repoId) {
    return (
      <Link to="/repos/$repoId" params={{ repoId: crumb.repoId }}>
        {crumb.label}
      </Link>
    )
  }
  return <Link to="/">{crumb.label}</Link>
}
