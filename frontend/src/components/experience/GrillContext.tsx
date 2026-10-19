import { createMemo, createSignal, For, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useCVTemplates } from "../../hooks/useCVTemplates";
import {
	useApplyContext,
	useGrillPrompt,
	usePreviewContext,
} from "../../hooks/useExperienceContext";
import type { ContextChange } from "../../types/experience";

const selectClass =
	"h-9 w-full rounded-md border border-border bg-surface px-3 text-sm text-foreground focus:border-primary focus:outline-none";

export function GrillContext(props: { onDone: () => void }) {
	const cvs = useCVTemplates();
	const prompt = useGrillPrompt();
	const preview = usePreviewContext();
	const apply = useApplyContext();
	const [selected, setSelected] = createSignal("");
	const [copied, setCopied] = createSignal(false);
	const [output, setOutput] = createSignal("");
	const [changes, setChanges] = createSignal<ContextChange[] | null>(null);

	const tabs = createMemo(() => (cvs.data ?? []).filter((cv) => cv.Visible));

	const copy = () => {
		const cv = tabs().find((t) => `${t.DocID}/${t.TabID}` === selected());
		setCopied(false);
		prompt.mutate(cv ? { docId: cv.DocID, tabId: cv.TabID } : null, {
			onSuccess: async (result) => {
				await navigator.clipboard.writeText(result.text);
				setCopied(true);
			},
		});
	};

	const run = () =>
		preview.mutate(output(), { onSuccess: (result) => setChanges(result) });

	const confirm = () => {
		const list = changes();
		if (!list?.length) return;
		apply.mutate(list, { onSuccess: props.onDone });
	};

	return (
		<div
			class="mb-4 space-y-5 rounded-xl border border-border bg-surface p-4"
			data-testid="grill-context"
		>
			<div class="space-y-2">
				<p class="text-sm font-medium text-foreground">1. Copy the prompt</p>
				<p class="text-sm text-muted">
					Paste it into Claude Code. It needs your <code>grilling</code> skill
					installed.
				</p>
				<div class="flex flex-wrap items-end gap-3">
					<div class="min-w-64 flex-1">
						<Label for="grill-tab">CV tab</Label>
						<select
							id="grill-tab"
							class={selectClass}
							value={selected()}
							onChange={(e) => setSelected(e.currentTarget.value)}
						>
							<option value="">No CV</option>
							<For each={tabs()}>
								{(cv) => (
									<option value={`${cv.DocID}/${cv.TabID}`}>
										{cv.SourceDoc} / {cv.Title}
									</option>
								)}
							</For>
						</select>
					</div>
					<Button onClick={copy} disabled={prompt.isPending}>
						{copied() ? "Copied" : "Copy grilling prompt"}
					</Button>
				</div>
				<Show when={prompt.error}>
					<p role="alert" class="text-sm text-destructive-strong">
						{prompt.error?.message}
					</p>
				</Show>
			</div>

			<div class="space-y-2">
				<Label for="grill-output">2. Paste Claude Code's output</Label>
				<Textarea
					id="grill-output"
					rows={6}
					value={output()}
					onInput={(e) => {
						setOutput(e.currentTarget.value);
						setChanges(null);
					}}
				/>
				<div class="flex gap-2">
					<Button
						onClick={run}
						disabled={!output().trim() || preview.isPending}
					>
						Preview context
					</Button>
					<Button variant="ghost" onClick={props.onDone}>
						Close
					</Button>
				</div>
				<Show when={preview.error}>
					<p role="alert" class="text-sm text-destructive-strong">
						{preview.error?.message}
					</p>
				</Show>
			</div>

			<Show when={changes()}>
				{(list) => (
					<div class="space-y-3" data-testid="context-preview">
						<Show
							when={list().length > 0}
							fallback={<p class="text-sm text-muted">No context to set.</p>}
						>
							<For each={list()}>
								{(c) => (
									<div
										class="space-y-1 rounded-lg border border-border p-3 text-sm"
										data-testid="context-change"
									>
										<p class="font-medium text-foreground">{c.text}</p>
										<p class="text-muted">Before: {c.before || "none"}</p>
										<p class="text-foreground">After: {c.after}</p>
									</div>
								)}
							</For>
							<Show when={apply.error}>
								<p role="alert" class="text-sm text-destructive-strong">
									{apply.error?.message}
								</p>
							</Show>
							<Button onClick={confirm} disabled={apply.isPending}>
								Apply context
							</Button>
						</Show>
					</div>
				)}
			</Show>
		</div>
	);
}
