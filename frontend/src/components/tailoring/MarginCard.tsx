import { For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { charsToSave, isSparse, type LineFit } from "@/lib/docLayout";
import { cn } from "@/lib/utils";
import { wordDiff } from "@/lib/wordDiff";
import { type DraftEditorState, SKILLS_CARD } from "../../hooks/useDraftEditor";

type Tone = { dot: string; text: string };

export function cardHeader(ed: DraftEditorState, key: string): Tone {
	if (key === SKILLS_CARD)
		return {
			dot: "bg-status-interview",
			text: `${ed.gaps().length} missing from your CV`,
		};
	if (ed.findingsFor(key).some((f) => f.severity === "block"))
		return { dot: "bg-destructive", text: "Blocking issue" };
	return ed.edited(key)
		? { dot: "bg-border-strong", text: "Your edit" }
		: { dot: "bg-status-interview", text: "Short last line" };
}

const SEVERITY_DOT = {
	block: "bg-destructive",
	warn: "bg-status-interview",
	info: "bg-border-strong",
};

function SkillGaps(props: { gaps: string[] }) {
	return (
		<>
			<p class="text-xs text-muted">
				The job asks for these and your CV never mentions them. Add one only if
				it is true.
			</p>
			<ul class="flex flex-col gap-1.5">
				<For each={props.gaps}>
					{(gap) => <li class="text-xs text-foreground">{gap}</li>}
				</For>
			</ul>
		</>
	);
}

function EditDiff(props: { from: string; to: string }) {
	return (
		<div>
			<p class="text-xs font-semibold text-muted">Your changes</p>
			<p class="mt-1 text-xs leading-relaxed" data-testid="card-diff">
				<For each={wordDiff(props.from, props.to)}>
					{(part) => (
						<span
							class={cn(
								part.op === "del" &&
									"text-faint line-through decoration-destructive/70",
								part.op === "add" && "bg-accent-subtle text-accent-text",
								part.op === "same" && "text-muted",
							)}
						>
							{part.text}
						</span>
					)}
				</For>
			</p>
		</div>
	);
}

function FitHint(props: {
	text: string;
	fit: LineFit;
	onFit?: (() => void) | undefined;
}) {
	return (
		<div class="flex items-center justify-between gap-2 rounded-lg bg-surface-muted px-2.5 py-2 ring-1 ring-border">
			<p class="text-xs text-muted" data-testid="card-fit">
				Last line is {Math.round(props.fit.lastLineFill * 100)}% full. Cut about{" "}
				{charsToSave(props.text, props.fit.lines, props.fit.lastLineFill)}{" "}
				characters to save a line.
			</p>
			<Show when={props.onFit}>
				{(onFit) => (
					<Button
						variant="outline"
						size="sm"
						class="shrink-0"
						onClick={() => onFit()()}
					>
						<Icon name="wand" size={13} />
						Fit
					</Button>
				)}
			</Show>
		</div>
	);
}

export function MarginCard(props: {
	cardKey: string;
	editor: DraftEditorState;
	fit: LineFit | undefined;
	active: boolean;
	editable: boolean;
	onOpen: () => void;
	onResolve: () => void;
	onFit?: ((slotId: string) => void) | undefined;
}) {
	const key = () => props.cardKey;
	const isSlot = () => key() !== SKILLS_CARD;
	const resolved = () => props.editor.isResolved(key());
	const edited = () => props.editor.edited(key());
	const header = () => cardHeader(props.editor, key());
	const sparseFit = () => (isSparse(props.fit) ? props.fit : undefined);
	return (
		<div
			data-testid="margin-card"
			data-card-key={key()}
			class={cn(
				"rounded-xl bg-surface text-sm transition-shadow duration-300",
				props.active
					? "shadow-xl ring-1 ring-accent-border"
					: "ring-1 ring-border",
				resolved() && !props.active && "opacity-60",
			)}
		>
			<button
				type="button"
				aria-expanded={props.active}
				class="flex w-full items-center gap-2 px-3.5 py-2.5 text-left"
				onClick={() => props.onOpen()}
			>
				<Show
					when={!resolved()}
					fallback={
						<Icon name="check" size={12} class="shrink-0 text-primary" />
					}
				>
					<span class={cn("size-2 shrink-0 rounded-full", header().dot)} />
				</Show>
				<span class="min-w-0 flex-1 truncate text-xs font-medium text-foreground">
					{header().text}
				</span>
				<Show when={props.fit}>
					{(fit) => (
						<span
							class={cn(
								"font-mono text-2xs tabular-nums",
								sparseFit() ? "text-status-interview" : "text-faint",
							)}
						>
							{fit().lines} {fit().lines === 1 ? "line" : "lines"}
						</span>
					)}
				</Show>
			</button>

			<Show when={props.active}>
				<div class="flex flex-col gap-3 border-t border-border px-3.5 pt-3 pb-3.5 animate-in fade-in duration-200">
					<Show
						when={isSlot()}
						fallback={<SkillGaps gaps={props.editor.gaps()} />}
					>
						<Show when={edited()}>
							<EditDiff
								from={props.editor.original(key())}
								to={props.editor.text(key())}
							/>
						</Show>
						<For each={props.editor.findingsFor(key())}>
							{(f) => (
								<p class="flex items-start gap-2 text-xs text-foreground">
									<span
										class={cn(
											"mt-1 size-1.5 shrink-0 rounded-full",
											SEVERITY_DOT[f.severity],
										)}
									/>
									{f.message}
								</p>
							)}
						</For>
						<Show when={sparseFit()}>
							{(fit) => (
								<FitHint
									text={props.editor.text(key())}
									fit={fit()}
									onFit={props.onFit && (() => props.onFit?.(key()))}
								/>
							)}
						</Show>
					</Show>

					<div class="flex items-center gap-2">
						<Button size="sm" onClick={() => props.onResolve()}>
							<Icon name="check" size={14} />
							{resolved() ? "Reopen" : "Resolve"}
						</Button>
						<Show when={isSlot() && props.editable && edited()}>
							<Button
								variant="ghost"
								size="sm"
								onClick={() => props.editor.undo(key())}
							>
								<Icon name="rotateCcw" size={14} />
								Undo my edits
							</Button>
						</Show>
					</div>
				</div>
			</Show>
		</div>
	);
}
