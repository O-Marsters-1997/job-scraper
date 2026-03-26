import { Outlet, createFileRoute, redirect } from '@tanstack/solid-router'
import Sidebar from '../components/Sidebar'
import { getMe } from '../api/auth'

export const Route = createFileRoute('/_auth')({
  beforeLoad: async () => {
    const res = await getMe()
    if (!res.ok) {
      throw redirect({ to: '/login' })
    }
  },
  component: AuthLayout,
})

function AuthLayout() {
  return (
    <div class="flex h-screen overflow-hidden">
      <Sidebar />
      <main class="flex-1 overflow-y-auto">
        <Outlet />
      </main>
    </div>
  )
}
