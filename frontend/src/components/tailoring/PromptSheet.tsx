import {
	createMemo,
	createSignal,
	For,
	onCleanup,
	onMount,
	Show,
	untrack,
} from "solid-js";
import { createStore, produce } from "solid-js/store";
import { streamPromptTry } from "@/api/tailoring";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { tailoringPrompt } from "@/lib/promptHandoff";
import {
	baseProfile,
	changedPaths,
	editedParts,
	mainText,
	missingReferences,
	partTexts,
	rawMessage,
	referencedPaths,
	sentRows,
	wordCount,
} from "@/lib/promptLab";
import { cn } from "@/lib/utils";
import { wordDiff } from "@/lib/wordDiff";
import type { PromptLab, PromptPart, PromptTry } from "@/types/promptLab";
import {
	usePromptLab,
	useResetPromptPart,
	useSavePromptPart,
	useUseProfileTry,
} from "../../hooks/useTailoring";
import { Marked } from "./ChangesDiff";
import { PasteReplyDialog } from "./PasteReplyDialog";

const MAX_STEER = 200;

function Tab(props: {
	path: string;
	active: boolean;
	unsaved?: boolean;
	flash?: boolean;
	onPick: () => void;
}) {
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
				props.flash && "ring-2 ring-primary motion-safe:animate-pulse",
			)}
			onClick={() => props.onPick()}
		>
			<Icon name="fileText" size={13} />
			{props.path}
			<Show when={props.unsaved}>
				<span class="rounded bg-accent-subtle px-1 text-2xs font-sans text-foreground">
					Unsaved
				</span>
			</Show>
		</button>
	);
}

function PartEditor(props: {
	draftId: string;
	generation: string;
	part: PromptPart;
	dirty: boolean;
	onEdit: (text: string) => void;
	onClear: () => void;
}) {
	const save = useSavePromptPart(() => props.draftId);
	const reset = useResetPromptPart(() => props.draftId);
	const key = () => ({ generation: props.generation, path: props.part.path });
	const status = () =>
		props.dirty
			? "Unsaved"
			: props.part.source === "yours"
				? "Yours"
				: "Default";
	return (
		<div class="flex flex-col gap-2">
			<Textarea
				data-testid="prompt-file"
				aria-label={props.part.path}
				rows={14}
				class="max-h-72 font-mono text-xs leading-relaxed"
				value={props.part.text}
				onInput={(e) => props.onEdit(e.currentTarget.value)}
			/>
			<div class="flex items-center gap-2">
				<span data-testid="prompt-file-status" class="text-xs text-muted">
					{status()}
				</span>
				<Show when={props.dirty}>
					<Button
						size="sm"
						disabled={save.isPending || props.part.text.trim() === ""}
						onClick={() =>
							save.mutate(
								{ ...key(), text: props.part.text },
								{ onSuccess: () => props.onClear() },
							)
						}
					>
						Save as mine
					</Button>
					<Button variant="ghost" size="sm" onClick={() => props.onClear()}>
						Discard changes
					</Button>
				</Show>
				<Show when={!props.dirty && props.part.source === "yours"}>
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

function Files(props: {
	draftId: string;
	lab: PromptLab;
	loaded: PromptLab;
	active: string;
	setActive: (path: string) => void;
	flash: boolean;
	onEdit: (path: string, text: string) => void;
	onClear: (path: string) => void;
}) {
	const active = () => props.active;
	const file = () =>
		[...props.lab.parts, ...props.lab.documents].find(
			(f) => f.path === active(),
		);
	const activePart = () => props.lab.parts.find((p) => p.path === active());
	const unsaved = (path: string) =>
		props.loaded.parts.find((p) => p.path === path)?.text !==
		props.lab.parts.find((p) => p.path === path)?.text;
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
							unsaved={unsaved(p.path)}
							flash={props.flash && active() === p.path}
							onPick={() => props.setActive(p.path)}
						/>
					)}
				</For>
				<span aria-hidden="true" class="mx-1 h-4 w-px bg-border" />
				<For each={props.lab.documents}>
					{(d) => (
						<Tab
							path={d.path}
							active={active() === d.path}
							onPick={() => props.setActive(d.path)}
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
						dirty={unsaved(p().path)}
						onEdit={(text) => props.onEdit(p().path, text)}
						onClear={() => props.onClear(p().path)}
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

function Brief(props: {
	lab: PromptLab;
	edits: Record<string, string>;
	tries: PromptTry[];
	onApply: (changes: { path: string; to: string }[]) => void;
}) {
	const [brief, setBrief] = createSignal("");
	const [note, setNote] = createSignal("");
	const [pasting, setPasting] = createSignal(false);
	const texts = () => partTexts(props.lab, props.edits);
	let timer: number | undefined;
	onCleanup(() => clearTimeout(timer));
	const copy = async () => {
		try {
			await navigator.clipboard.writeText(
				tailoringPrompt({
					brief: brief(),
					lab: props.lab,
					texts: texts(),
					tries: props.tries,
				}),
			);
			setNote("Copied, paste it into Claude Code");
		} catch {
			setNote("Could not copy. Check clipboard permission.");
		}
		clearTimeout(timer);
		timer = window.setTimeout(() => setNote(""), 2500);
	};
	return (
		<section class="flex flex-col gap-2">
			<label for="prompt-brief" class="text-sm font-semibold text-foreground">
				Brief
			</label>
			<Textarea
				id="prompt-brief"
				rows={3}
				placeholder="What should change? e.g. the Profile sounds generic."
				value={brief()}
				onInput={(e) => setBrief(e.currentTarget.value)}
			/>
			<div class="flex items-center gap-2">
				<Button size="sm" onClick={copy}>
					Copy tailoring prompt
				</Button>
				<Button size="sm" variant="ghost" onClick={() => setPasting(true)}>
					Paste reply
				</Button>
				<output class="text-xs text-muted">{note()}</output>
			</div>
			<PasteReplyDialog
				open={pasting()}
				texts={texts()}
				onOpenChange={setPasting}
				onApply={props.onApply}
			/>
		</section>
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

function TryCard(props: {
	t: PromptTry;
	base: string;
	budgetWords: number;
	baseLines: number;
	latest: boolean;
	using: boolean;
	onUse: () => void;
}) {
	const words = () => wordCount(props.t.text);
	const over = () => words() > props.budgetWords;
	const overLines = () => (props.t.lines ?? 0) > props.baseLines;
	const reduceMotion = () =>
		window.matchMedia("(prefers-reduced-motion: reduce)").matches;
	return (
		<li
			ref={(el) =>
				queueMicrotask(() =>
					el.scrollIntoView({
						behavior: reduceMotion() ? "auto" : "smooth",
						block: "nearest",
					}),
				)
			}
			data-testid="prompt-try"
			class="flex flex-col gap-2 rounded-lg border border-border px-3 py-2.5"
		>
			<div class="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted">
				<span class="font-medium text-foreground">Try {props.t.n}</span>
				<Show
					when={props.t.changed.length > 0}
					fallback={<span>defaults</span>}
				>
					<For each={props.t.changed}>
						{(p) => (
							<code class="rounded border border-border px-1.5 font-mono text-2xs">
								{p}
							</code>
						)}
					</For>
				</Show>
				<Show when={props.t.steer}>
					<span class="italic">"{props.t.steer}"</span>
				</Show>
				<span class="ml-auto tabular-nums">${props.t.cost.toFixed(3)}</span>
			</div>
			<Show when={props.t.error}>
				<p role="alert" class="text-xs text-destructive-strong">
					{props.t.error}
				</p>
			</Show>
			<Show when={!props.t.streaming} fallback={<StreamingText t={props.t} />}>
				<p class="text-sm leading-relaxed">
					<Marked ops={wordDiff(props.base, props.t.text)} />
				</p>
			</Show>
			<div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
				<span
					class={cn(
						"tabular-nums",
						over() ? "text-destructive-strong" : "text-muted",
					)}
				>
					{words()} / {props.budgetWords} words
				</span>
				<span
					aria-hidden="true"
					class="h-1 w-24 overflow-hidden rounded-full bg-surface-muted"
				>
					<span
						class={cn(
							"block h-full transition-[width]",
							over() ? "bg-destructive" : "bg-primary",
						)}
						style={{
							width: `${Math.min(100, (words() / Math.max(1, props.budgetWords)) * 100)}%`,
						}}
					/>
				</span>
				<Show when={props.t.lines !== null}>
					<span class={overLines() ? "text-destructive-strong" : "text-muted"}>
						{props.t.lines} {props.t.lines === 1 ? "line" : "lines"}
						{overLines() && `, over the ${props.baseLines}-line budget`}
					</span>
				</Show>
			</div>
			<For each={props.t.findings}>
				{(f) => (
					<p role="alert" class="text-xs text-destructive-strong">
						{f.message}
					</p>
				)}
			</For>
			<Show when={props.latest && !props.t.streaming && !props.t.error}>
				<div class="flex flex-wrap items-center gap-2 text-xs">
					<Show
						when={!props.t.used}
						fallback={
							<output class="text-foreground">
								In your Draft. Saved as a Profile Attempt.
							</output>
						}
					>
						<Button
							size="sm"
							variant="outline"
							disabled={props.t.findings.length > 0}
							loading={props.using}
							onClick={() => props.onUse()}
						>
							Use this text
						</Button>
						<Show when={props.t.findings.length > 0}>
							<span class="text-muted">
								Blocked: fix the findings above first.
							</span>
						</Show>
					</Show>
					<Show when={props.t.useError}>
						<span role="alert" class="text-destructive-strong">
							{props.t.useError}
						</span>
					</Show>
				</div>
			</Show>
		</li>
	);
}

function StreamingText(props: { t: PromptTry }) {
	return (
		<p class="text-sm leading-relaxed text-muted" aria-busy="true">
			{props.t.text || "Waiting for the model…"}
		</p>
	);
}

function Workbench(props: {
	draftId: string;
	loaded: PromptLab;
	measureLines: (text: string) => number;
}) {
	const [showMessage, setShowMessage] = createSignal(false);
	const [edited, setEdited] = createStore<Record<string, string>>({});
	const [tries, setTries] = createStore<PromptTry[]>([]);
	const [steer, setSteer] = createSignal("");
	const [running, setRunning] = createSignal(false);
	const [active, setActive] = createSignal(untrack(() => props.loaded.main));
	const [flash, setFlash] = createSignal(false);
	const [applied, setApplied] = createSignal<string[]>([]);
	let flashTimer: number | undefined;
	onCleanup(() => clearTimeout(flashTimer));
	const apply = (changes: { path: string; to: string }[]) => {
		const first = changes[0];
		if (!first) return;
		setEdited(Object.fromEntries(changes.map((c) => [c.path, c.to])));
		setApplied(changes.map((c) => c.path));
		setActive(first.path);
		setFlash(true);
		clearTimeout(flashTimer);
		flashTimer = window.setTimeout(() => setFlash(false), 2000);
	};
	const ctrl = new AbortController();
	onMount(() => {
		const onKey = (e: KeyboardEvent) => {
			if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) void run();
		};
		window.addEventListener("keydown", onKey);
		onCleanup(() => window.removeEventListener("keydown", onKey));
	});
	onCleanup(() => ctrl.abort());

	const lab = createMemo<PromptLab>(() => ({
		...props.loaded,
		parts: props.loaded.parts.map((p) => ({
			...p,
			text: edited[p.path] ?? p.text,
		})),
	}));
	const base = () => baseProfile(props.loaded);
	const budgetWords = createMemo(() => wordCount(base()));
	const baseLines = createMemo(() => props.measureLines(base()));
	const sessionCost = () => tries.reduce((sum, t) => sum + t.cost, 0);

	const use = useUseProfileTry(() => props.draftId);
	async function useText(t: PromptTry) {
		const patch = (p: Partial<PromptTry>) => setTries((x) => x.n === t.n, p);
		patch({ useError: undefined });
		try {
			await use.mutateAsync({
				text: t.text,
				steer: t.steer,
				parts: t.parts,
			});
			patch({ used: true });
		} catch (err) {
			patch({
				useError:
					err instanceof Error ? err.message : "Could not use this text",
			});
		}
	}

	async function run() {
		if (running()) return;
		const n = tries.length + 1;
		const req = {
			parts: editedParts(props.loaded, edited),
			steer: steer().trim(),
		};
		setTries(
			produce((all) => {
				all.unshift({
					n,
					text: "",
					streaming: true,
					findings: [],
					lines: null,
					changed: changedPaths(props.loaded, edited),
					steer: req.steer,
					parts: req.parts,
					cost: 0,
				});
			}),
		);
		const patch = (p: Partial<PromptTry>) => setTries((t) => t.n === n, p);
		setRunning(true);
		try {
			const done = await streamPromptTry(
				props.draftId,
				req,
				(d) =>
					setTries(
						(t) => t.n === n,
						"text",
						(s) => s + d,
					),
				ctrl.signal,
			);
			patch({
				text: done.text,
				streaming: false,
				findings: done.findings.filter((f) => f.severity === "block"),
				lines: props.measureLines(done.text),
				cost: done.cost,
			});
		} catch (err) {
			if (ctrl.signal.aborted) return;
			patch({
				streaming: false,
				error: err instanceof Error ? err.message : "Try failed",
			});
		} finally {
			setRunning(false);
		}
	}

	return (
		<>
			<div class="scroll-slim flex flex-1 flex-col gap-4 overflow-y-auto px-6 py-5">
				<Brief lab={lab()} edits={edited} tries={[...tries]} onApply={apply} />
				<Show when={applied().length > 0}>
					<output class="block text-xs text-foreground">
						Applied Claude Code's changes to {applied().join(", ")}. Try to see
						the effect.
					</output>
				</Show>
				<Files
					draftId={props.draftId}
					lab={lab()}
					loaded={props.loaded}
					active={active()}
					setActive={setActive}
					flash={flash()}
					onEdit={(path, text) => setEdited(path, text)}
					onClear={(path) =>
						setEdited(
							produce((all) => {
								delete all[path];
							}),
						)
					}
				/>
				<Show when={tries.length > 0}>
					<section class="flex flex-col gap-2" aria-label="Tries">
						<div class="flex items-center justify-between text-xs text-muted">
							<h3 class="font-medium text-foreground">Tries</h3>
							<span class="tabular-nums">
								Session cost ${sessionCost().toFixed(3)}
							</span>
						</div>
						<ol class="flex flex-col gap-2">
							<For each={tries}>
								{(t) => (
									<TryCard
										t={t}
										base={base()}
										budgetWords={budgetWords()}
										baseLines={baseLines()}
										latest={t.n === tries.length}
										using={use.isPending}
										onUse={() => void useText(t)}
									/>
								)}
							</For>
						</ol>
					</section>
				</Show>
				<button
					type="button"
					aria-expanded={showMessage()}
					class="flex items-center gap-1.5 self-start text-xs font-medium text-muted hover:text-foreground"
					onClick={() => setShowMessage((v) => !v)}
				>
					<Icon
						name={showMessage() ? "chevronDown" : "chevronRight"}
						size={12}
					/>
					What the model receives
				</button>
				<Show when={showMessage()}>
					<Received lab={lab()} />
				</Show>
			</div>
			<footer class="flex items-center gap-2 border-t border-border px-6 py-3">
				<Input
					aria-label="Steer"
					placeholder="Steer this try (optional)"
					maxLength={MAX_STEER}
					class="min-w-0 flex-1"
					value={steer()}
					onInput={(e) => setSteer(e.currentTarget.value)}
				/>
				<Button loading={running()} onClick={() => void run()}>
					Try
					<kbd class="ml-1.5 font-sans text-2xs opacity-70">⌘↵</kbd>
				</Button>
			</footer>
		</>
	);
}

export function PromptSheet(props: {
	draftId: string;
	measureLines: (text: string) => number;
	onClose: () => void;
}) {
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
						The files the Redraft reads. Edit them and Try, or save your own
						version of a file.
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
				{(l) => (
					<Workbench
						draftId={props.draftId}
						loaded={l()}
						measureLines={props.measureLines}
					/>
				)}
			</QueryBoundary>
		</aside>
	);
}
