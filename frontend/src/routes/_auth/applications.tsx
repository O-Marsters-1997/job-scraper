import { createFileRoute } from "@tanstack/solid-router"
import { createSignal, For, Show } from "solid-js"
import { useApplications, useUpdateApplication, useDeleteApplication } from "../../hooks/useApplications"
import { useApplicationStatuses } from "../../hooks/useApplicationStatuses"
import type { ApplicationWithDetails } from "../../types/application"

export const Route = createFileRoute("/_auth/applications")({
  component: ApplicationsPage,
})

function ApplicationsPage() {
  const [statusFilter, setStatusFilter] = createSignal<string | undefined>(undefined)
  const query = useApplications(statusFilter)
  const statusesQuery = useApplicationStatuses()
  const updateMutation = useUpdateApplication()
  const deleteMutation = useDeleteApplication()

  const [editingApp, setEditingApp] = createSignal<ApplicationWithDetails | null>(null)
  const [editStatusId, setEditStatusId] = createSignal("")
  const [editNotes, setEditNotes] = createSignal("")
  const [editAppliedAt, setEditAppliedAt] = createSignal("")
  const [editSalary, setEditSalary] = createSignal("")

  const openEdit = (app: ApplicationWithDetails) => {
    setEditingApp(app)
    setEditStatusId(app.StatusID)
    setEditNotes(app.Notes)
    setEditAppliedAt(app.AppliedAt ?? "")
    setEditSalary(app.SalaryInfo)
  }

  const closeEdit = () => setEditingApp(null)

  const handleSave = async () => {
    const app = editingApp()
    if (!app) return
    await updateMutation.mutateAsync({
      id: app.ID,
      data: {
        status_id: editStatusId() || undefined,
        notes: editNotes(),
        applied_at: editAppliedAt() || null,
        salary_info: editSalary(),
      },
    })
    closeEdit()
  }

  const handleDelete = async (id: string) => {
    await deleteMutation.mutateAsync(id)
  }

  const statusColour = (app: ApplicationWithDetails) => {
    if (app.StatusColour) return app.StatusColour
    const status = statusesQuery.data?.find((s) => s.ID === app.StatusID)
    return status?.Colour ?? "#64748b"
  }

  return (
    <div class="px-6 pb-12 pt-8">
      <div class="mb-8">
        <p class="island-kicker mb-2">Tracking</p>
        <h1 class="display-title text-4xl font-bold text-[var(--sea-ink)] sm:text-5xl">
          Applications
        </h1>
      </div>

      {/* Filter bar */}
      <div class="mb-6 flex items-center gap-3 flex-wrap">
        <button
          type="button"
          onClick={() => setStatusFilter(undefined)}
          class={`rounded-full border px-3 py-1 text-xs font-medium transition ${
            statusFilter() === undefined
              ? "border-[rgba(50,143,151,0.5)] bg-[rgba(79,184,178,0.28)] text-[var(--sea-ink)]"
              : "border-[var(--line)] text-[var(--sea-ink-soft)] hover:border-[rgba(50,143,151,0.3)] hover:text-[var(--sea-ink)]"
          }`}
        >
          All
        </button>
        <For each={statusesQuery.data}>
          {(s) => (
            <button
              type="button"
              onClick={() => setStatusFilter(s.ID)}
              class={`rounded-full border px-3 py-1 text-xs font-medium transition ${
                statusFilter() === s.ID
                  ? "border-[rgba(50,143,151,0.5)] bg-[rgba(79,184,178,0.28)] text-[var(--sea-ink)]"
                  : "border-[var(--line)] text-[var(--sea-ink-soft)] hover:border-[rgba(50,143,151,0.3)] hover:text-[var(--sea-ink)]"
              }`}
            >
              <span
                class="mr-1.5 inline-block h-2 w-2 rounded-full"
                style={{ background: s.Colour }}
              />
              {s.Name}
            </button>
          )}
        </For>
      </div>

      <Show when={query.isPending}>
        <p class="text-sm text-[var(--sea-ink-soft)]">Loading…</p>
      </Show>

      <Show when={query.isSuccess}>
        <Show
          when={(query.data?.length ?? 0) > 0}
          fallback={
            <div class="island-shell rounded-xl p-8 text-center">
              <p class="text-sm text-[var(--sea-ink-soft)]">No applications yet. Track a job from the Jobs page.</p>
            </div>
          }
        >
          <div class="island-shell rounded-xl divide-y divide-[var(--line)]">
            <For each={query.data}>
              {(app) => (
                <div class="flex items-center gap-4 px-4 py-3">
                  <div class="flex-1 min-w-0">
                    <p class="truncate text-sm font-medium text-[var(--sea-ink)]">{app.JobTitle}</p>
                    <p class="text-xs text-[var(--sea-ink-soft)]">{app.JobCompanySlug}{app.JobLocation ? ` · ${app.JobLocation}` : ""}</p>
                  </div>
                  <Show when={app.StatusName}>
                    <span
                      class="shrink-0 rounded-full px-2.5 py-0.5 text-xs font-medium text-white"
                      style={{ background: statusColour(app) }}
                    >
                      {app.StatusName}
                    </span>
                  </Show>
                  <Show when={app.AppliedAt}>
                    <span class="shrink-0 text-xs text-[var(--sea-ink-soft)]">{app.AppliedAt}</span>
                  </Show>
                  <button
                    type="button"
                    onClick={() => openEdit(app)}
                    class="shrink-0 text-xs text-[var(--lagoon-deep)] hover:underline"
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    onClick={() => handleDelete(app.ID)}
                    disabled={deleteMutation.isPending}
                    class="shrink-0 text-xs text-red-500 hover:text-red-700 disabled:opacity-50"
                  >
                    Delete
                  </button>
                </div>
              )}
            </For>
          </div>
        </Show>
      </Show>

      {/* Edit modal */}
      <Show when={editingApp()}>
        <div
          class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm"
          onClick={(e) => e.target === e.currentTarget && closeEdit()}
        >
          <div class="island-shell w-full max-w-md rounded-2xl p-6">
            <h2 class="mb-4 text-base font-semibold text-[var(--sea-ink)]">
              Edit Application
            </h2>
            <p class="mb-4 text-sm text-[var(--sea-ink-soft)]">{editingApp()?.JobTitle}</p>

            <div class="space-y-4">
              <div>
                <label class="mb-1 block text-xs font-medium text-[var(--sea-ink-soft)]">Status</label>
                <select
                  class="w-full rounded border border-[var(--line)] bg-[var(--surface)] px-3 py-2 text-sm text-[var(--sea-ink)] focus:outline-none"
                  value={editStatusId()}
                  onChange={(e) => setEditStatusId(e.currentTarget.value)}
                >
                  <option value="">— No status —</option>
                  <For each={statusesQuery.data}>
                    {(s) => <option value={s.ID}>{s.Name}</option>}
                  </For>
                </select>
              </div>

              <div>
                <label class="mb-1 block text-xs font-medium text-[var(--sea-ink-soft)]">Applied date</label>
                <input
                  type="date"
                  class="w-full rounded border border-[var(--line)] bg-[var(--surface)] px-3 py-2 text-sm text-[var(--sea-ink)] focus:outline-none"
                  value={editAppliedAt()}
                  onInput={(e) => setEditAppliedAt(e.currentTarget.value)}
                />
              </div>

              <div>
                <label class="mb-1 block text-xs font-medium text-[var(--sea-ink-soft)]">Salary / comp</label>
                <input
                  type="text"
                  class="w-full rounded border border-[var(--line)] bg-[var(--surface)] px-3 py-2 text-sm text-[var(--sea-ink)] focus:outline-none"
                  placeholder="e.g. £80,000"
                  value={editSalary()}
                  onInput={(e) => setEditSalary(e.currentTarget.value)}
                />
              </div>

              <div>
                <label class="mb-1 block text-xs font-medium text-[var(--sea-ink-soft)]">Notes</label>
                <textarea
                  class="w-full rounded border border-[var(--line)] bg-[var(--surface)] px-3 py-2 text-sm text-[var(--sea-ink)] focus:outline-none resize-none"
                  rows={3}
                  placeholder="Any notes about this application…"
                  value={editNotes()}
                  onInput={(e) => setEditNotes(e.currentTarget.value)}
                />
              </div>
            </div>

            <div class="mt-6 flex justify-end gap-3">
              <button
                type="button"
                onClick={closeEdit}
                class="rounded-full border border-[var(--line)] px-4 py-1.5 text-sm text-[var(--sea-ink-soft)] transition hover:text-[var(--sea-ink)]"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSave}
                disabled={updateMutation.isPending}
                class="rounded-full border border-[rgba(50,143,151,0.3)] bg-[rgba(79,184,178,0.14)] px-4 py-1.5 text-sm font-semibold text-[var(--lagoon-deep)] transition hover:-translate-y-0.5 hover:bg-[rgba(79,184,178,0.24)] disabled:opacity-50"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      </Show>
    </div>
  )
}
