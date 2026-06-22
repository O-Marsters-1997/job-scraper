import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import {
  useApplicationStatuses,
  useCreateApplicationStatus,
  useUpdateApplicationStatus,
  useDeleteApplicationStatus,
} from "../../../hooks/useApplicationStatuses";
import { STATUS_PALETTE } from "@/lib/status";
import { SkeletonList } from "@/components/ui/skeleton";
import type { ApplicationStatus } from "../../../types/applicationStatus";

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
  const [newColour, setNewColour] = createSignal(STATUS_PALETTE[0].hex);

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
    setNewColour(STATUS_PALETTE[0].hex);
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
    <div class="max-w-2xl px-7 py-6">
      <div class="mb-5">
        <h1 class="text-lg font-bold tracking-tight text-foreground">
          Application statuses
        </h1>
        <p class="mt-0.5 text-xs text-faint">
          Custom stages for your pipeline, each with its own colour
        </p>
      </div>

      <Show when={deleteError()}>
        <div class="mb-4 rounded-lg border border-destructive/30 bg-destructive-subtle px-4 py-3 text-sm text-destructive-strong">
          {deleteError()}
        </div>
      </Show>

      <Show when={query.isPending}>
        <SkeletonList rows={5} />
      </Show>

      <Show when={query.isSuccess}>
        <div class="divide-y divide-border overflow-hidden rounded-xl border border-border bg-surface">
          <For each={query.data}>
            {(status) => (
              <div class="flex items-center gap-3 px-4 py-3">
                <Show
                  when={editingId() === status.ID}
                  fallback={
                    <>
                      <span
                        class="h-3 w-3 shrink-0 rounded-full"
                        style={{ background: status.Colour }}
                      />
                      <span class="flex-1 text-sm font-medium text-foreground">
                        {status.Name}
                      </span>
                      <button
                        type="button"
                        onClick={() => startEdit(status)}
                        class="rounded px-2 py-1 text-xs font-medium text-muted transition hover:bg-surface-muted hover:text-foreground"
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        onClick={() => handleDelete(status.ID)}
                        class="rounded px-2 py-1 text-xs font-medium text-destructive transition hover:bg-destructive-subtle"
                      >
                        Delete
                      </button>
                    </>
                  }
                >
                  <div class="flex flex-1 items-center gap-2">
                    <div class="flex gap-1">
                      <For each={STATUS_PALETTE}>
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
                                  ? "var(--color-foreground)"
                                  : "transparent",
                            }}
                          />
                        )}
                      </For>
                    </div>
                    <input
                      class="flex-1 rounded-md border border-border bg-surface px-2 py-1 text-sm text-foreground focus:border-primary focus:outline-none"
                      value={editName()}
                      onInput={(e) => setEditName(e.currentTarget.value)}
                    />
                  </div>
                  <button
                    type="button"
                    onClick={handleUpdate}
                    disabled={updateMutation.isPending}
                    class="rounded px-2 py-1 text-xs font-medium text-primary transition hover:bg-accent-subtle disabled:opacity-50"
                  >
                    Save
                  </button>
                  <button
                    type="button"
                    onClick={() => setEditingId(null)}
                    class="rounded px-2 py-1 text-xs font-medium text-muted transition hover:bg-surface-muted hover:text-foreground"
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
                <For each={STATUS_PALETTE}>
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
                            ? "var(--color-foreground)"
                            : "transparent",
                      }}
                    />
                  )}
                </For>
              </div>
              <input
                class="flex-1 rounded-md border border-border bg-surface px-2 py-1 text-sm text-foreground focus:border-primary focus:outline-none"
                placeholder="Status name"
                value={newName()}
                onInput={(e) => setNewName(e.currentTarget.value)}
                onKeyDown={(e) => e.key === "Enter" && handleAdd()}
              />
              <button
                type="button"
                onClick={handleAdd}
                disabled={createMutation.isPending || !newName().trim()}
                class="rounded px-2 py-1 text-xs font-medium text-primary transition hover:bg-accent-subtle disabled:opacity-50"
              >
                Add
              </button>
              <button
                type="button"
                onClick={() => {
                  setShowAdd(false);
                  setNewName("");
                }}
                class="rounded px-2 py-1 text-xs font-medium text-muted transition hover:bg-surface-muted hover:text-foreground"
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
            class="mt-4 inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground transition hover:bg-primary-hover"
          >
            <svg
              aria-hidden="true"
              width="12"
              height="12"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
              stroke-linecap="round"
            >
              <line x1="12" y1="5" x2="12" y2="19" />
              <line x1="5" y1="12" x2="19" y2="12" />
            </svg>
            Add status
          </button>
        </Show>
        <p class="mt-3 text-xs text-faint">
          Changes take effect immediately. Deleting a status does not remove it
          from existing applications.
        </p>
      </Show>
    </div>
  );
}
