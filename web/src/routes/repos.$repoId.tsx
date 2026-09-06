import { Outlet, createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/repos/$repoId')({
  component: () => <Outlet />,
})
