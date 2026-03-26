import { createFileRoute, useNavigate } from '@tanstack/solid-router'
import { createSignal } from 'solid-js'
import { login } from '../api/auth'

export const Route = createFileRoute('/login')({
  component: LoginPage,
})

function LoginPage() {
  const navigate = useNavigate()
  const [username, setUsername] = createSignal('')
  const [password, setPassword] = createSignal('')
  const [error, setError] = createSignal('')
  const [loading, setLoading] = createSignal(false)

  const handleSubmit = async (e: SubmitEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await login(username(), password())
      if (res.ok) {
        navigate({ to: '/jobs', search: { page: 1 } })
      } else {
        setError('Invalid username or password.')
      }
    } catch {
      setError('Could not reach the server. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div class="flex min-h-screen items-center justify-center bg-[var(--surface)]">
      <div class="island-shell w-full max-w-sm rounded-2xl p-8">
        <div class="mb-8 flex flex-col items-center gap-3">
          <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-[rgba(79,184,178,0.18)] text-[var(--lagoon-deep)]">
            <svg
              aria-hidden="true"
              width="22"
              height="22"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <rect x="2" y="7" width="20" height="14" rx="2" />
              <path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2" />
            </svg>
          </div>
          <div class="text-center">
            <p class="island-kicker mb-1">Welcome back</p>
            <h1 class="text-xl font-bold text-[var(--sea-ink)]">Job Scraper</h1>
          </div>
        </div>

        <form onSubmit={handleSubmit} class="flex flex-col gap-4">
          <div class="flex flex-col gap-1.5">
            <label
              for="username"
              class="text-xs font-semibold uppercase tracking-wide text-[var(--sea-ink-soft)]"
            >
              Username
            </label>
            <input
              id="username"
              type="text"
              autocomplete="username"
              required
              value={username()}
              onInput={(e) => setUsername(e.currentTarget.value)}
              class="rounded-lg border border-[var(--line)] bg-[var(--surface-strong)] px-3 py-2 text-sm text-[var(--sea-ink)] placeholder-[var(--sea-ink-soft)] outline-none transition focus:border-[rgba(50,143,151,0.5)] focus:ring-2 focus:ring-[rgba(79,184,178,0.2)]"
              placeholder="alice"
            />
          </div>

          <div class="flex flex-col gap-1.5">
            <label
              for="password"
              class="text-xs font-semibold uppercase tracking-wide text-[var(--sea-ink-soft)]"
            >
              Password
            </label>
            <input
              id="password"
              type="password"
              autocomplete="current-password"
              required
              value={password()}
              onInput={(e) => setPassword(e.currentTarget.value)}
              class="rounded-lg border border-[var(--line)] bg-[var(--surface-strong)] px-3 py-2 text-sm text-[var(--sea-ink)] placeholder-[var(--sea-ink-soft)] outline-none transition focus:border-[rgba(50,143,151,0.5)] focus:ring-2 focus:ring-[rgba(79,184,178,0.2)]"
              placeholder="••••••••"
            />
          </div>

          {error() && (
            <p class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-600">
              {error()}
            </p>
          )}

          <button
            type="submit"
            disabled={loading()}
            class="mt-2 inline-flex items-center justify-center rounded-full border border-[rgba(50,143,151,0.3)] bg-[rgba(79,184,178,0.14)] px-4 py-2 text-sm font-semibold text-[var(--lagoon-deep)] transition hover:-translate-y-0.5 hover:bg-[rgba(79,184,178,0.24)] disabled:pointer-events-none disabled:opacity-50"
          >
            {loading() ? 'Signing in…' : 'Sign in'}
          </button>
        </form>
      </div>
    </div>
  )
}
