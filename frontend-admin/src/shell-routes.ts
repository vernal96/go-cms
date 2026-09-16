// Shared by routing and plugin validation; components stay in the shell.
export const shellRoutes = {
  root: { name: 'shell.root', path: '/admin' },
  dashboard: { name: 'shell.dashboard', path: '/admin/dashboard' },
  profile: { name: 'shell.profile', path: '/admin/profile' },
} as const
