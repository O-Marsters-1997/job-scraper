import {
	createEffect,
	For,
	type JSX,
	onCleanup,
	onMount,
	Show,
} from "solid-js";
import { createStore, reconcile } from "solid-js/store";
import { resolveFont } from "@/lib/docFonts";
import {
	type BorderDraw,
	borderDraws,
	hasEndTab,
	isSpacer,
	isSparse,
	type LineFit,
	linePt,
	splitAtTab,
	trimParagraphEnd,
} from "@/lib/docLayout";
import { cn } from "@/lib/utils";
import type { WordOp } from "@/lib/wordDiff";
import type {
	DraftLayout,
	LayoutBlock,
	LayoutBorder,
	LayoutRun,
} from "@/types/tailoring";

const INK = "#000000";
const LINK_INK = "#1155cc";
const PAPER = "#ffffff";

export const PX_PER_PT = 4 / 3;
export const DEFAULT_BODY_LINE_PT = 15;
const DEFAULT_FONT_PT = 11;
const LINE_COUNT_GUTTER_PT = 14;
const SAME_ROW_TOLERANCE_PX = 2;

export type PageMetrics = {
	contentPt: number;
	availablePt: number;
	bodyLinePt: number;
	fits: Record<string, LineFit>;
};

export type PageEditor = {
	editable: boolean;
	showCounts: boolean;
	active: string | undefined;
	hovered: string | undefined;
	hasCard: (slotId: string) => boolean;
	onHover: (slotId: string, over: boolean) => void;
	text: (slotId: string) => string;
	label: (slotId: string) => string;
	diffFor: (slotId: string) => WordOp[] | null;
	onInput: (slotId: string, text: string) => void;
	onFocus: (slotId: string) => void;
	onDismissSuggestion: (slotId: string) => void;
	onBlur: (slotId: string) => void;
	onEditSkills?: (() => void) | undefined;
};

const ALIGN: Record<string, JSX.CSSProperties["text-align"]> = {
	center: "center",
	right: "right",
	justify: "justify",
};

function dominantRun(runs: LayoutRun[]): LayoutRun | undefined {
	return runs.reduce<LayoutRun | undefined>(
		(best, r) => (!best || r.size > best.size ? r : best),
		undefined,
	);
}

const borderCss = (b: LayoutBorder | null) =>
	b ? `${b.width}pt ${b.dash} ${b.color || INK}` : undefined;

const borderPadding = (b: LayoutBorder | null) =>
	b ? `${b.padding}pt` : undefined;

function blockStyle(
	b: LayoutBlock,
	lead: LayoutRun | undefined,
	draw: BorderDraw,
	spacer: boolean,
): JSX.CSSProperties {
	const font = resolveFont(lead?.font ?? "");
	const size = lead?.size ?? DEFAULT_FONT_PT;
	const lineHeight = linePt(1, font.ratio, b.lineSpacing);
	const top = draw.top ? b.borderTop : null;
	const bottom = draw.bottom ? b.borderBottom : null;
	return {
		"font-family": font.css,
		"font-size": `${size}pt`,
		"line-height": `${lineHeight}`,
		height: spacer ? `${size * lineHeight}pt` : undefined,
		"margin-top": `${b.spaceAbove}pt`,
		"margin-bottom": `${b.spaceBelow}pt`,
		"padding-left": `${b.indentStart}pt`,
		"text-indent": b.bullet
			? undefined
			: `${b.indentFirstLine - b.indentStart}pt`,
		"text-align": ALIGN[b.align],
		"border-top": borderCss(top),
		"border-bottom": borderCss(bottom),
		"padding-top": borderPadding(top),
		"padding-bottom": borderPadding(bottom),
	};
}

function runStyle(r: LayoutRun): JSX.CSSProperties {
	return {
		"font-family": resolveFont(r.font).css,
		"font-size": `${r.size}pt`,
		"font-weight": r.bold ? 700 : 400,
		"font-style": r.italic ? "italic" : undefined,
		"text-decoration": r.underline || r.link ? "underline" : undefined,
		color: r.color || (r.link ? LINK_INK : undefined),
	};
}

const singleLine = (text: string) => text.replace(/\s*\n\s*/g, " ");

function Runs(props: { runs: LayoutRun[] }) {
	return (
		<For each={props.runs}>
			{(r) => (
				<Show
					when={r.link}
					fallback={<span style={runStyle(r)}>{r.text}</span>}
				>
					<a href={r.link} target="_blank" rel="noreferrer" style={runStyle(r)}>
						{r.text}
					</a>
				</Show>
			)}
		</For>
	);
}

function StaticLine(props: { runs: LayoutRun[]; endTab: boolean }) {
	const split = () => (props.endTab ? splitAtTab(props.runs) : null);
	return (
		<Show when={split()} fallback={<Runs runs={props.runs} />}>
			{(parts) => (
				<span class="flex justify-between gap-2">
					<span class="min-w-0">
						<Runs runs={parts()[0]} />
					</span>
					<span class="whitespace-nowrap">
						<Runs runs={parts()[1]} />
					</span>
				</span>
			)}
		</Show>
	);
}

function measureFit(el: HTMLElement): LineFit {
	const lineHeight = Number.parseFloat(getComputedStyle(el).lineHeight);
	const lines = Math.max(
		1,
		Math.round(el.offsetHeight / (lineHeight || el.offsetHeight)),
	);
	const range = document.createRange();
	range.selectNodeContents(el);
	const rects = [...range.getClientRects()].filter((r) => r.width > 0);
	const last = rects.at(-1);
	if (!last) return { lines, lastLineFill: 0 };
	const row = rects.filter(
		(r) => Math.abs(r.top - last.top) < SAME_ROW_TOLERANCE_PX,
	);
	const left = Math.min(...row.map((r) => r.left));
	const right = Math.max(...row.map((r) => r.right));
	return {
		lines,
		lastLineFill: (right - left) / el.getBoundingClientRect().width,
	};
}

function EditableLine(props: {
	slotId: string;
	editor: PageEditor;
	style: JSX.CSSProperties;
	ref: (el: HTMLElement) => void;
}) {
	let el: HTMLElement | undefined;
	const text = () => props.editor.text(props.slotId);
	const diff = () => props.editor.diffFor(props.slotId);
	createEffect(() => {
		const value = text();
		if (el && el.textContent !== value) el.textContent = value;
	});
	return (
		<span class="relative block" style={props.style}>
			<Show when={diff()}>
				{(parts) => (
					<span data-testid="suggestion-diff" class="relative block">
						<For each={parts()}>
							{(part) => (
								<span
									class={cn(
										part.op === "del" &&
											"line-through opacity-40 decoration-destructive",
										part.op === "add" && "bg-accent-subtle text-accent-text",
									)}
								>
									{part.text}
								</span>
							)}
						</For>
					</span>
				)}
			</Show>
			{/* biome-ignore lint/a11y/useSemanticElements: an <input> or <textarea> cannot reproduce the CV's own line wrapping, which contenteditable does */}
			<span
				ref={(e) => {
					el = e;
					e.textContent = text();
					props.ref(e);
				}}
				role="textbox"
				aria-label={props.editor.label(props.slotId)}
				aria-multiline="false"
				aria-readonly={!props.editor.editable}
				tabIndex={0}
				contentEditable={
					props.editor.editable && !diff() ? "plaintext-only" : false
				}
				spellcheck={false}
				class={cn(
					"relative block cursor-text caret-primary outline-none selection:bg-primary/30",
					diff() && "pointer-events-none invisible absolute inset-x-0 top-0",
				)}
				onInput={(e) =>
					props.editor.onInput(
						props.slotId,
						singleLine(e.currentTarget.textContent ?? ""),
					)
				}
				onPaste={(e) => {
					e.preventDefault();
					const pasted = e.clipboardData?.getData("text/plain") ?? "";
					document.execCommand("insertText", false, singleLine(pasted));
				}}
				onKeyDown={(e) => {
					if (e.key === "Enter") e.preventDefault();
					if (e.key === "Escape") e.currentTarget.blur();
				}}
				onFocus={() => props.editor.onFocus(props.slotId)}
				onBlur={() => props.editor.onBlur(props.slotId)}
			/>
		</span>
	);
}

function caretAt(line: HTMLElement, x: number, y: number) {
	const box = line.getBoundingClientRect();
	const px = Math.min(Math.max(x, box.left + 1), box.right - 1);
	const py = Math.min(Math.max(y, box.top + 1), box.bottom - 1);
	const pos = document.caretPositionFromPoint?.(px, py);
	if (pos && line.contains(pos.offsetNode)) return pos;
	const range = document.caretRangeFromPoint?.(px, py);
	if (range && line.contains(range.startContainer))
		return { offsetNode: range.startContainer, offset: range.startOffset };
	return undefined;
}

function claimClick(e: MouseEvent, ed: PageEditor, slotId: string) {
	const block = e.currentTarget;
	if (!(block instanceof HTMLElement) || e.button !== 0 || !ed.editable) return;
	const line = block.querySelector<HTMLElement>('[role="textbox"]');
	if (!line) return;
	const suggested = ed.diffFor(slotId) !== null;
	if (!suggested && e.target instanceof Node && line.contains(e.target)) return;
	e.preventDefault();
	if (suggested) ed.onDismissSuggestion(slotId);
	const { clientX, clientY } = e;
	queueMicrotask(() => {
		line.focus({ preventScroll: true });
		const sel = getSelection();
		if (!sel) return;
		const at = caretAt(line, clientX, clientY);
		const range = document.createRange();
		if (at) {
			range.setStart(at.offsetNode, at.offset);
			range.collapse(true);
		} else {
			range.selectNodeContents(line);
			range.collapse(false);
		}
		sel.removeAllRanges();
		sel.addRange(range);
	});
}

function LineCount(props: {
	fit: LineFit;
	page: DraftLayout["page"];
	indentStart: number;
}) {
	return (
		<span
			aria-hidden="true"
			data-testid="line-count"
			title={`${props.fit.lines} lines, last line ${Math.round(props.fit.lastLineFill * 100)}% full`}
			class={cn(
				"pointer-events-none absolute top-0 text-right font-mono text-2xs tabular-nums",
				isSparse(props.fit) ? "text-status-interview" : "text-faint",
			)}
			style={{
				left: `${-props.page.marginLeft - props.indentStart}pt`,
				width: `${Math.max(props.page.marginLeft - LINE_COUNT_GUTTER_PT, 0)}pt`,
			}}
		>
			{props.fit.lines}
		</span>
	);
}

function Block(props: {
	block: LayoutBlock;
	draw: BorderDraw;
	page: DraftLayout["page"];
	editor: PageEditor | undefined;
	fit: LineFit | undefined;
	register: (slotId: string, el: HTMLElement) => void;
}) {
	const spacer = () => isSpacer(props.block);
	const lead = () => dominantRun(props.block.runs);
	const leadStyle = () => {
		const run = lead();
		return run ? runStyle(run) : {};
	};
	const editor = () =>
		props.block.slotId && !props.block.section ? props.editor : undefined;
	const onEditSkills = () =>
		props.block.section === "skills" ? props.editor?.onEditSkills : undefined;
	return (
		<div
			data-slot-id={props.block.slotId || undefined}
			data-section={props.block.section || undefined}
			data-skill-line={props.block.skillLine ?? undefined}
			class={cn(
				"relative whitespace-pre-wrap [font-kerning:normal] [font-variant-ligatures:none] [tab-size:36pt]",
				editor()?.editable && "cursor-text",
			)}
			on:mouseenter={() => {
				const ed = editor();
				if (ed?.hasCard(props.block.slotId))
					ed.onHover(props.block.slotId, true);
			}}
			on:mouseleave={() => editor()?.onHover(props.block.slotId, false)}
			on:mousedown={(e) => {
				const ed = editor();
				if (ed) claimClick(e, ed, props.block.slotId);
			}}
			style={blockStyle(props.block, lead(), props.draw, spacer())}
		>
			<Show when={onEditSkills()}>
				{(open) => (
					<button
						type="button"
						aria-label="Edit skills"
						title="Edit skills"
						class="absolute inset-0 z-10 cursor-pointer rounded-sm hover:bg-accent-subtle/60 focus-visible:outline-2 focus-visible:outline-primary"
						onClick={() => open()()}
					/>
				)}
			</Show>
			<Show when={editor()}>
				{(ed) => (
					<>
						<Show when={ed().active === props.block.slotId}>
							<span
								aria-hidden="true"
								class="absolute -left-2 inset-y-0 w-1 rounded-full bg-primary"
							/>
						</Show>
						<Show
							when={
								ed().hasCard(props.block.slotId) ||
								ed().active === props.block.slotId
							}
						>
							<span
								aria-hidden="true"
								data-testid="line-anchor"
								class={cn(
									"pointer-events-none absolute inset-0 rounded-sm transition-colors",
									ed().active === props.block.slotId ||
										ed().hovered === props.block.slotId
										? "bg-primary/20"
										: "bg-accent-subtle",
								)}
							/>
						</Show>
						<Show when={ed().showCounts && props.fit}>
							{(fit) => (
								<LineCount
									fit={fit()}
									page={props.page}
									indentStart={props.block.indentStart}
								/>
							)}
						</Show>
					</>
				)}
			</Show>
			<Show when={props.block.bullet}>
				{(bullet) => (
					<span
						aria-hidden="true"
						class="pointer-events-none absolute"
						style={{
							left: `${Math.min(props.block.indentFirstLine, Math.max(props.block.indentStart - bullet().size, 0))}pt`,
							"font-size": bullet().size > 0 ? `${bullet().size}pt` : undefined,
						}}
					>
						{bullet().glyph}
					</span>
				)}
			</Show>
			<Show when={!spacer()}>
				<Show
					when={editor()}
					fallback={
						<StaticLine
							runs={trimParagraphEnd(props.block.runs)}
							endTab={hasEndTab(props.block)}
						/>
					}
				>
					{(ed) => (
						<EditableLine
							slotId={props.block.slotId}
							editor={ed()}
							style={leadStyle()}
							ref={(el) => props.register(props.block.slotId, el)}
						/>
					)}
				</Show>
			</Show>
		</div>
	);
}

export function DocPage(props: {
	layout: DraftLayout;
	editor?: PageEditor;
	onMetrics?: (metrics: PageMetrics) => void;
	onMeasurer?: (lines: (slotId: string, text: string) => number) => void;
	class?: string;
}) {
	const page = () => props.layout.page;
	const draws = () => borderDraws(props.layout.blocks);
	const lines = new Map<string, HTMLElement>();
	const [fits, setFits] = createStore<Record<string, LineFit>>({});
	let content: HTMLDivElement | undefined;

	const measure = () => {
		if (!content) return;
		const next: Record<string, LineFit> = {};
		let bodyLinePx = 0;
		for (const [id, el] of lines) {
			if (!el.isConnected) continue;
			next[id] = measureFit(el);
			bodyLinePx ||= Number.parseFloat(getComputedStyle(el).lineHeight);
		}
		setFits(reconcile(next));
		props.onMetrics?.({
			contentPt: content.offsetHeight / PX_PER_PT,
			availablePt: page().height - page().marginTop - page().marginBottom,
			bodyLinePt: bodyLinePx ? bodyLinePx / PX_PER_PT : DEFAULT_BODY_LINE_PT,
			fits: next,
		});
	};
	const remeasure = () => queueMicrotask(measure);

	const measureLines = (slotId: string, text: string) => {
		const el = lines.get(slotId);
		const block = el?.closest<HTMLElement>("[data-slot-id]");
		if (!block?.parentElement) return 1;
		const clone = block.cloneNode(true);
		if (!(clone instanceof HTMLElement)) return 1;
		clone.removeAttribute("data-slot-id");
		clone.querySelector('[data-testid="suggestion-diff"]')?.remove();
		clone.style.position = "absolute";
		clone.style.visibility = "hidden";
		clone.style.width = `${block.offsetWidth}px`;
		const line = clone.querySelector<HTMLElement>('[role="textbox"]');
		if (!line) return 1;
		line.className = "relative block";
		line.textContent = text;
		block.parentElement.append(clone);
		const { lines: count } = measureFit(line);
		clone.remove();
		return count;
	};

	onMount(() => {
		props.onMeasurer?.(measureLines);
		document.fonts?.ready.then(remeasure);
		const ro = new ResizeObserver(remeasure);
		if (content) ro.observe(content);
		onCleanup(() => ro.disconnect());
	});
	createEffect(() => {
		for (const b of props.layout.blocks)
			if (b.slotId) props.editor?.text(b.slotId);
		remeasure();
	});

	return (
		<div
			class={cn("relative shrink-0 shadow-sm ring-1 ring-border", props.class)}
			style={{
				width: `${page().width}pt`,
				"min-height": `${page().height}pt`,
				"padding-top": `${page().marginTop}pt`,
				"padding-bottom": `${page().marginBottom}pt`,
				"padding-left": `${page().marginLeft}pt`,
				"padding-right": `${page().marginRight}pt`,
				background: PAPER,
				color: INK,
			}}
		>
			<div ref={content} class="flow-root">
				<For each={props.layout.blocks}>
					{(block, i) => (
						<Block
							block={block}
							page={page()}
							editor={props.editor}
							fit={fits[block.slotId]}
							draw={draws()[i()] ?? { top: true, bottom: true }}
							register={(id, el) => {
								lines.set(id, el);
								remeasure();
							}}
						/>
					)}
				</For>
			</div>
			<div
				aria-hidden="true"
				data-testid="page-end"
				class="pointer-events-none absolute inset-x-0 border-t border-dashed border-destructive/50"
				style={{ top: `${page().height - page().marginBottom}pt` }}
			>
				<span class="absolute right-2 bottom-0.5 font-mono text-2xs text-destructive-strong">
					page 1 ends
				</span>
			</div>
		</div>
	);
}
