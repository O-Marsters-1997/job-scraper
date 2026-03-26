import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import {
  useApplicationStatuses,
  useCreateApplicationStatus,
  useUpdateApplicationStatus,
  useDeleteApplicationStatus,
} from "../../../hooks/useApplicationStatuses";
import type { ApplicationStatus } from "../../../types/applicationStatus";

const PALETTE = [
  { hex: "#6366f1", label: "Indigo" },
  { hex: "#22c55e", label: "Green" },
  { hex: "#ef4444", label: "Red" },
  { hex: "#f59e0b", label: "Amber" },
  { hex: "#3b82f6", label: "Blue" },
  { hex: "#a855f7", label: "Purple" },
  { hex: "#64748b", label: "Slate" },
  { hex: "#ec4899", label: "Pink" },
];

export const Route = createFileRoute("/_auth/settings/statuses")({
  component: StatusesPage,
});

function StatusesPage() {
  const query = useApplicationStatuses();
  const createMutation = useCreateApplicationStatus();
  const updateMutation = useUpdateApplicationStatus();
  const deleteMutation = useDeleteApplicationStatus();

  const [showAdd, setShowAdd] = createSignal(false);
  const [newName, setNewName] = createSignal("");
  const [newColour, setNewColour] = createSignal(PALETTE[0].hex);

  const [editingId, setEditingId] = createSignal<string | null>(null);
  const [editName, setEditName] = createSignal("");
  const [editColour, setEditColour] = createSignal("");

  const [deleteError, setDeleteError] = createSignal<string | null>(null);

  const handleAdd = async () => {
    if (!newName().trim()) return;
    await createMutation.mutateAsync({
      name: newName().trim(),
      colour: newColour(),
    });
    setNewName("");
    setNewColour(PALETTE[0].hex);
    setShowAdd(false);
  };

  const startEdit = (s: ApplicationStatus) => {
    setEditingId(s.ID);
    setEditName(s.Name);
    setEditColour(s.Colour);
  };

  const handleUpdate = async () => {
    const id = editingId();
    if (!id || !editName().trim()) return;
    await updateMutation.mutateAsync({
      id,
      name: editName().trim(),
      colour: editColour(),
    });
    setEditingId(null);
  };

  const handleDelete = async (id: string) => {
    setDeleteError(null);
    const result = await deleteMutation.mutateAsync(id);
    if (result.count !== undefined) {
      setDeleteError(
        `Cannot delete: ${result.count} application${result.count !== 1 ? "s" : ""} use this status. Reassign them first.`,
      );
    }
  };

  return (
    <div class="px-6 pb-12 pt-8 max-w-2xl">
      <div class="mb-8">
        <p class="island-kicker mb-2">Settings</p>
        <h1 class="display-title text-4xl font-bold text-[var(--sea-ink)] sm:text-5xl">
          Application Statuses
        </h1>
        <p class="mt-2 text-sm text-[var(--sea-ink-soft)]">
          Customise the statuses used to track your job applications.
        </p>
      </div>

      <Show when={deleteError()}>
        <div class="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {deleteError()}
        </div>
      </Show>

      <Show when={query.isPending}>
        <p class="text-sm text-[var(--sea-ink-soft)]">Loading…</p>
      </Show>

      <Show when={query.isSuccess}>
        <div class="island-shell rounded-xl divide-y divide-[var(--line)]">
          <For each={query.data}>
            {(status) => (
              <div class="flex items-center gap-3 px-4 py-3">
                <Show
                  when={editingId() === status.ID}
                  fallback={
                    <>
                      <span
                        class="h-4 w-4 shrink-0 rounded-full"
                        style={{ background: status.Colour }}
                      />
                      <span class="flex-1 text-sm text-[var(--sea-ink)]">
                        {status.Name}
                      </span>
                      <button
                        type="button"
                        onClick={() => startEdit(status)}
                        class="text-xs text-[var(--sea-ink-soft)] hover:text-[var(--sea-ink)]"
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        onClick={() => handleDelete(status.ID)}
                        class="text-xs text-red-500 hover:text-red-700"
                      >
                        Delete
                      </button>
                    </>
                  }
                >
                  <div class="flex flex-1 items-center gap-2">
                    <div class="flex gap-1">
                      <For each={PALETTE}>
                        {(p) => (
                          <button
                            type="button"
                            title={p.label}
                            onClick={() => setEditColour(p.hex)}
                            class="h-5 w-5 rounded-full border-2 transition"
                            style={{
                              background: p.hex,
                              "border-color":
                                editColour() === p.hex
                                  ? "var(--sea-ink)"
                                  : "transparent",
                            }}
                          />
                        )}
                      </For>
                    </div>
                    <input
                      class="flex-1 rounded border border-[var(--line)] bg-[var(--surface)] px-2 py-1 text-sm text-[var(--sea-ink)] focus:outline-none"
                      value={editName()}
                      onInput={(e) => setEditName(e.currentTarget.value)}
                    />
                  </div>
                  <button
                    type="button"
                    onClick={handleUpdate}
                    disabled={updateMutation.isPending}
                    class="text-xs font-medium text-[var(--lagoon-deep)] hover:underline disabled:opacity-50"
                  >
                    Save
                  </button>
                  <button
                    type="button"
                    onClick={() => setEditingId(null)}
                    class="text-xs text-[var(--sea-ink-soft)] hover:text-[var(--sea-ink)]"
                  >
                    Cancel
                  </button>
                </Show>
              </div>
            )}
          </For>

          <Show when={showAdd()}>
            <div class="flex items-center gap-2 px-4 py-3">
              <div class="flex gap-1">
                <For each={PALETTE}>
                  {(p) => (
                    <button
                      type="button"
                      title={p.label}
                      onClick={() => setNewColour(p.hex)}
                      class="h-5 w-5 rounded-full border-2 transition"
                      style={{
                        background: p.hex,
                        "border-color":
                          newColour() === p.hex
                            ? "var(--sea-ink)"
                            : "transparent",
                      }}
                    />
                  )}
                </For>
              </div>
              <input
                class="flex-1 rounded border border-[var(--line)] bg-[var(--surface)] px-2 py-1 text-sm text-[var(--sea-ink)] focus:outline-none"
                placeholder="Status name"
                value={newName()}
                onInput={(e) => setNewName(e.currentTarget.value)}
                onKeyDown={(e) => e.key === "Enter" && handleAdd()}
              />
              <button
                type="button"
                onClick={handleAdd}
                disabled={createMutation.isPending || !newName().trim()}
                class="text-xs font-medium text-[var(--lagoon-deep)] hover:underline disabled:opacity-50"
              >
                Add
              </button>
              <button
                type="button"
                onClick={() => {
                  setShowAdd(false);
                  setNewName("");
                }}
                class="text-xs text-[var(--sea-ink-soft)] hover:text-[var(--sea-ink)]"
              >
                Cancel
              </button>
            </div>
          </Show>
        </div>

        <Show when={!showAdd()}>
          <button
            type="button"
            onClick={() => setShowAdd(true)}
            class="mt-4 inline-flex items-center gap-1 rounded-full border border-[rgba(50,143,151,0.3)] bg-[rgba(79,184,178,0.14)] px-4 py-1.5 text-sm font-semibold text-[var(--lagoon-deep)] transition hover:-translate-y-0.5 hover:bg-[rgba(79,184,178,0.24)]"
          >
            + Add status
          </button>
        </Show>
      </Show>
    </div>
  );
}
