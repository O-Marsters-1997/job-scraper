import { createSignal, For, onCleanup, onMount, Show, untrack } from "solid-js";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
	mainText,
	missingReferences,
	rawMessage,
	referencedPaths,
	sentRows,
	wordCount,
} from "@/lib/promptLab";
import { cn } from "@/lib/utils";
import type { PromptLab, PromptPart } from "@/types/promptLab";
import {
	usePromptLab,
	useResetPromptPart,
	useSavePromptPart,
} from "../../hooks/useTailoring";

function Tab(props: { path: string; active: boolean; onPick: () => void }) {
	return (
		<button
			type="button"
			role="tab"
			aria-selected={props.active}
			class={cn(
				"flex items-center gap-1.5 rounded-md px-2.5 py-1.5 font-mono text-xs transition-colors",
				props.active
					? "bg-accent-subtle text-foreground"
					: "text-muted hover:bg-surface-muted hover:text-foreground",
			)}
			onClick={() => props.onPick()}
		>
			<Icon name="fileText" size={13} />
			{props.path}
		</button>
	);
}

function PartEditor(props: {
	draftId: string;
	generation: string;
	part: PromptPart;
}) {
	const [edited, setEdited] = createSignal<string>();
	const text = () => edited() ?? props.part.text;
	const dirty = () => text() !== props.part.text;
	const save = useSavePromptPart(() => props.draftId);
	const reset = useResetPromptPart(() => props.draftId);
	const key = () => ({ generation: props.generation, path: props.part.path });
	const status = () =>
		dirty() ? "Unsaved" : props.part.source === "yours" ? "Yours" : "Default";
	return (
		<div class="flex flex-col gap-2">
			<Textarea
				data-testid="prompt-file"
				aria-label={props.part.path}
				rows={14}
				class="max-h-72 font-mono text-xs leading-relaxed"
				value={text()}
				onInput={(e) => setEdited(e.currentTarget.value)}
			/>
			<div class="flex items-center gap-2">
				<span data-testid="prompt-file-status" class="text-xs text-muted">
					{status()}
				</span>
				<Show when={dirty()}>
					<Button
						size="sm"
						disabled={save.isPending || text().trim() === ""}
						onClick={() =>
							save.mutate(
								{ ...key(), text: text() },
								{ onSuccess: () => setEdited(undefined) },
							)
						}
					>
						Save as mine
					</Button>
					<Button
						variant="ghost"
						size="sm"
						onClick={() => setEdited(undefined)}
					>
						Discard changes
					</Button>
				</Show>
				<Show when={!dirty() && props.part.source === "yours"}>
					<Button
						variant="ghost"
						size="sm"
						disabled={reset.isPending}
						onClick={() => reset.mutate(key())}
					>
						Use the default
					</Button>
				</Show>
			</div>
			<Show when={save.error ?? reset.error}>
				{(err) => (
					<p role="alert" class="text-xs text-destructive-strong">
						{err().message}
					</p>
				)}
			</Show>
		</div>
	);
}

function Files(props: { lab: PromptLab; draftId: string }) {
	const [active, setActive] = createSignal(untrack(() => props.lab.main));
	const file = () =>
		[...props.lab.parts, ...props.lab.documents].find(
			(f) => f.path === active(),
		);
	const activePart = () => props.lab.parts.find((p) => p.path === active());
	const references = () =>
		referencedPaths(mainText(props.lab)).filter((p) => p !== props.lab.main);
	return (
		<div class="flex flex-col gap-2.5">
			<div role="tablist" class="flex flex-wrap items-center gap-1">
				<For each={props.lab.parts}>
					{(p) => (
						<Tab
							path={p.path}
							active={active() === p.path}
							onPick={() => setActive(p.path)}
						/>
					)}
				</For>
				<span aria-hidden="true" class="mx-1 h-4 w-px bg-border" />
				<For each={props.lab.documents}>
					{(d) => (
						<Tab
							path={d.path}
							active={active() === d.path}
							onPick={() => setActive(d.path)}
						/>
					)}
				</For>
			</div>
			<Show when={activePart()}>
				{(p) => (
					<PartEditor
						draftId={props.draftId}
						generation={props.lab.generation}
						part={p()}
					/>
				)}
			</Show>
			<Show when={!activePart() && file()}>
				{(f) => (
					<pre
						data-testid="prompt-file"
						class="max-h-72 overflow-auto whitespace-pre-wrap rounded-md border border-border bg-surface-muted px-3 py-2 font-mono text-xs leading-relaxed text-foreground"
					>
						{f().text}
					</pre>
				)}
			</Show>
			<Show when={active() === props.lab.main && references().length > 0}>
				<p class="flex flex-wrap items-center gap-1.5 text-xs text-muted">
					References
					<For each={references()}>
						{(p) => (
							<code
								class={cn(
									"rounded border px-1.5 font-mono text-2xs",
									missingReferences(props.lab).includes(p)
										? "border-destructive text-destructive-strong"
										: "border-border",
								)}
							>
								{p}
								{missingReferences(props.lab).includes(p) && " (not visible)"}
							</code>
						)}
					</For>
				</p>
			</Show>
		</div>
	);
}

function Received(props: { lab: PromptLab }) {
	const [open, setOpen] = createSignal<string>();
	const [raw, setRaw] = createSignal(false);
	return (
		<div class="flex flex-col gap-2">
			<div class="flex items-center justify-between gap-3">
				<p class="text-xs text-muted">
					Sent in this order. FastTrack's shared system prompt comes first and
					isn't shown.
				</p>
				<button
					type="button"
					class="shrink-0 text-xs font-medium text-muted underline-offset-4 hover:text-foreground hover:underline"
					onClick={() => setRaw((v) => !v)}
				>
					{raw() ? "Show as files" : "Show raw message"}
				</button>
			</div>
			<Show
				when={!raw()}
				fallback={
					<pre class="max-h-80 overflow-auto whitespace-pre-wrap rounded-lg border border-border bg-surface-muted p-3 font-mono text-2xs leading-relaxed text-muted">
						{rawMessage(props.lab)}
					</pre>
				}
			>
				<ol class="divide-y divide-border overflow-hidden rounded-lg border border-border">
					<For each={sentRows(props.lab)}>
						{(r) => (
							<li>
								<button
									type="button"
									aria-expanded={open() === r.path}
									class="flex w-full items-center gap-2 px-3 py-2 text-left transition-colors hover:bg-surface-muted"
									onClick={() =>
										setOpen((o) => (o === r.path ? undefined : r.path))
									}
								>
									<Icon
										name={open() === r.path ? "chevronDown" : "chevronRight"}
										size={12}
										class="text-faint"
									/>
									<code class="font-mono text-xs text-foreground">
										{r.path}
									</code>
									<span class="ml-auto text-2xs text-faint">
										{r.source} · {wordCount(r.text)} words
									</span>
								</button>
								<Show when={open() === r.path}>
									<pre class="max-h-56 overflow-auto whitespace-pre-wrap border-t border-border bg-surface-muted px-3 py-2.5 font-mono text-2xs leading-relaxed text-muted">
										{r.text}
									</pre>
								</Show>
							</li>
						)}
					</For>
				</ol>
			</Show>
		</div>
	);
}

function Body(props: { lab: PromptLab; draftId: string }) {
	const [showMessage, setShowMessage] = createSignal(false);
	return (
		<div class="scroll-slim flex flex-1 flex-col gap-4 overflow-y-auto px-6 py-5">
			<Files lab={props.lab} draftId={props.draftId} />
			<button
				type="button"
				aria-expanded={showMessage()}
				class="flex items-center gap-1.5 self-start text-xs font-medium text-muted hover:text-foreground"
				onClick={() => setShowMessage((v) => !v)}
			>
				<Icon name={showMessage() ? "chevronDown" : "chevronRight"} size={12} />
				What the model receives
			</button>
			<Show when={showMessage()}>
				<Received lab={props.lab} />
			</Show>
		</div>
	);
}

export function PromptSheet(props: { draftId: string; onClose: () => void }) {
	const lab = usePromptLab(() => props.draftId);
	let heading: HTMLHeadingElement | undefined;
	onMount(() => {
		heading?.focus();
		const onKey = (e: KeyboardEvent) => e.key === "Escape" && props.onClose();
		window.addEventListener("keydown", onKey);
		onCleanup(() => window.removeEventListener("keydown", onKey));
	});
	return (
		<aside
			aria-labelledby="prompt-sheet-title"
			class="fixed inset-y-0 right-0 z-40 flex w-full flex-col border-l border-border bg-surface shadow-2xl sm:w-[34rem] motion-safe:animate-in motion-safe:slide-in-from-right-6 motion-safe:fade-in motion-safe:duration-200"
		>
			<header class="flex items-start justify-between gap-3 border-b border-border px-6 py-4">
				<div>
					<h2
						id="prompt-sheet-title"
						ref={heading}
						tabIndex={-1}
						class="text-base font-semibold tracking-tight text-foreground outline-none"
					>
						Tune the Profile prompt
					</h2>
					<p class="mt-0.5 text-xs text-muted">
						The files the Redraft reads. Save your own version of a file.
					</p>
				</div>
				<Button
					variant="ghost"
					size="icon-sm"
					aria-label="Close"
					title="Close (Esc)"
					onClick={() => props.onClose()}
				>
					<Icon name="x" size={16} />
				</Button>
			</header>
			<QueryBoundary query={lab} fallbackRows={4}>
				{(l) => <Body lab={l()} draftId={props.draftId} />}
			</QueryBoundary>
		</aside>
	);
}
