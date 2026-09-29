import { createMemo, createSignal, For, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useCVTemplates } from "../../hooks/useCVTemplates";
import {
	useImportExperience,
	usePreviewExperienceImport,
} from "../../hooks/useExperienceImport";
import type { ImportPosition } from "../../types/experience";

export function ImportFromCV(props: { onDone: () => void }) {
	const cvs = useCVTemplates();
	const preview = usePreviewExperienceImport();
	const commit = useImportExperience();
	const [selected, setSelected] = createSignal("");
	const [draft, setDraft] = createSignal<ImportPosition[] | null>(null);

	const tabs = createMemo(() => (cvs.data ?? []).filter((cv) => cv.Visible));

	const load = () => {
		const cv = tabs().find((t) => `${t.DocID}/${t.TabID}` === selected());
		if (!cv) return;
		preview.mutate(
			{ docId: cv.DocID, tabId: cv.TabID },
			{ onSuccess: (positions) => setDraft(positions) },
		);
	};

	const patch = (i: number, change: Partial<ImportPosition>) =>
		setDraft((d) => d?.map((p, j) => (j === i ? { ...p, ...change } : p)) ?? d);

	const remove = (i: number) =>
		setDraft((d) => d?.filter((_, j) => j !== i) ?? d);

	const complete = () =>
		(draft() ?? []).every((p) => p.employer.trim() && p.title.trim());

	const confirm = () => {
		const positions = draft();
		if (!positions?.length) return;
		commit.mutate(positions, { onSuccess: props.onDone });
	};

	return (
		<div
			class="mb-4 space-y-4 rounded-xl border border-border bg-surface p-4"
			data-testid="import-from-cv"
		>
			<Show when={draft() === null}>
				<div class="flex flex-wrap items-end gap-3">
					<div class="min-w-64 flex-1">
						<Label for="import-tab">CV tab</Label>
						<select
							id="import-tab"
							class="h-9 w-full rounded-md border border-border bg-surface px-3 text-sm text-foreground focus:border-primary focus:outline-none"
							value={selected()}
							onChange={(e) => setSelected(e.currentTarget.value)}
						>
							<option value="">Choose a tracked CV tab</option>
							<For each={tabs()}>
								{(cv) => (
									<option value={`${cv.DocID}/${cv.TabID}`}>
										{cv.SourceDoc} / {cv.Title}
									</option>
								)}
							</For>
						</select>
					</div>
					<Button onClick={load} disabled={!selected() || preview.isPending}>
						Preview import
					</Button>
					<Button variant="ghost" onClick={props.onDone}>
						Cancel
					</Button>
				</div>
				<Show when={preview.error}>
					<p role="alert" class="text-sm text-destructive-strong">
						{preview.error?.message}
					</p>
				</Show>
			</Show>

			<Show when={draft()}>
				{(positions) => (
					<>
						<p class="text-sm text-muted">
							Review before importing. Nothing is saved until you confirm, and
							imported positions are added alongside your existing ones.
						</p>
						<Show when={positions().length === 0}>
							<p class="text-sm text-muted">No positions found in that tab.</p>
						</Show>
						<For each={positions()}>
							{(p, i) => (
								<div
									class="space-y-3 rounded-lg border border-border p-3"
									data-testid="import-position"
								>
									<Show when={p.employerExists}>
										<p class="text-sm font-medium text-accent-text">
											Your Bank already has a position at this employer.
											Importing adds another.
										</p>
									</Show>
									<div class="grid gap-3 sm:grid-cols-2">
										<div>
											<Label for={`import-employer-${i()}`}>Employer</Label>
											<Input
												id={`import-employer-${i()}`}
												value={p.employer}
												onInput={(e) =>
													patch(i(), { employer: e.currentTarget.value })
												}
											/>
										</div>
										<div>
											<Label for={`import-title-${i()}`}>Title</Label>
											<Input
												id={`import-title-${i()}`}
												value={p.title}
												onInput={(e) =>
													patch(i(), { title: e.currentTarget.value })
												}
											/>
										</div>
										<div>
											<Label for={`import-start-${i()}`}>Start date</Label>
											<Input
												id={`import-start-${i()}`}
												type="date"
												value={p.startDate ?? ""}
												onInput={(e) =>
													patch(i(), {
														startDate: e.currentTarget.value || null,
													})
												}
											/>
										</div>
										<div>
											<Label for={`import-end-${i()}`}>
												End date (blank if current)
											</Label>
											<Input
												id={`import-end-${i()}`}
												type="date"
												value={p.endDate ?? ""}
												onInput={(e) =>
													patch(i(), { endDate: e.currentTarget.value || null })
												}
											/>
										</div>
									</div>
									<ul class="list-disc space-y-1 pl-5 text-sm text-foreground">
										<For each={p.achievements}>{(text) => <li>{text}</li>}</For>
									</ul>
									<Button variant="ghost" onClick={() => remove(i())}>
										Skip this position
									</Button>
								</div>
							)}
						</For>
						<Show when={commit.error}>
							<p role="alert" class="text-sm text-destructive-strong">
								{commit.error?.message}
							</p>
						</Show>
						<div class="flex gap-2">
							<Button
								onClick={confirm}
								disabled={
									positions().length === 0 || !complete() || commit.isPending
								}
							>
								Confirm import
							</Button>
							<Button variant="ghost" onClick={props.onDone}>
								Cancel
							</Button>
						</div>
					</>
				)}
			</Show>
		</div>
	);
}
