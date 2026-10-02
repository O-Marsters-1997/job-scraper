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
	linePt,
	SPARSE_LAST_LINE,
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

const PX_PER_PT = 4 / 3;

export type LineFit = { lines: number; lastLineFill: number };

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
	text: (slotId: string) => string;
	label: (slotId: string) => string;
	diffFor: (slotId: string) => WordOp[] | null;
	onInput: (slotId: string, text: string) => void;
	onFocus: (slotId: string) => void;
	onBlur: (slotId: string) => void;
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

function borderCss(b: LayoutBorder) {
	return `${b.width}pt ${b.dash} ${b.color || INK}`;
}

function blockStyle(
	b: LayoutBlock,
	draw: BorderDraw,
	spacer: boolean,
): JSX.CSSProperties {
	const lead = dominantRun(b.runs);
	const font = resolveFont(lead?.font ?? "");
	const size = lead?.size ?? 11;
	const hanging = b.indentFirstLine - b.indentStart;
	return {
		"font-family": font.css,
		"font-size": `${size}pt`,
		"line-height": `${(font.ratio * (b.lineSpacing > 0 ? b.lineSpacing : 100)) / 100}`,
		height: spacer ? `${linePt(size, font.ratio, b.lineSpacing)}pt` : undefined,
		"margin-top": `${b.spaceAbove}pt`,
		"margin-bottom": `${b.spaceBelow}pt`,
		"padding-left": `${b.indentStart}pt`,
		"text-indent": b.bullet ? undefined : `${hanging}pt`,
		"text-align": ALIGN[b.align],
		"border-top": b.borderTop && draw.top ? borderCss(b.borderTop) : undefined,
		"border-bottom":
			b.borderBottom && draw.bottom ? borderCss(b.borderBottom) : undefined,
		"padding-top":
			b.borderTop && draw.top ? `${b.borderTop.padding}pt` : undefined,
		"padding-bottom":
			b.borderBottom && draw.bottom ? `${b.borderBottom.padding}pt` : undefined,
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
	const row = rects.filter((r) => Math.abs(r.top - last.top) < 2);
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
	runStyle: JSX.CSSProperties;
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
		<span class="relative block" style={props.runStyle}>
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
					"relative block cursor-text caret-primary outline-none selection:bg-accent-border",
					diff() && "pointer-events-none invisible absolute inset-x-0 top-0",
				)}
				onInput={(e) =>
					props.editor.onInput(
						props.slotId,
						(e.currentTarget.textContent ?? "").replace(/\s*\n\s*/g, " "),
					)
				}
				onPaste={(e) => {
					e.preventDefault();
					const pasted = e.clipboardData?.getData("text/plain") ?? "";
					document.execCommand(
						"insertText",
						false,
						pasted.replace(/\s*\n\s*/g, " "),
					);
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

function Block(props: {
	block: LayoutBlock;
	draw: BorderDraw;
	page: DraftLayout["page"];
	editor: PageEditor | undefined;
	fit: LineFit | undefined;
	register: (slotId: string, el: HTMLElement) => void;
}) {
	const spacer = () => isSpacer(props.block);
	const runs = () => trimParagraphEnd(props.block.runs);
	const split = () => (hasEndTab(props.block) ? splitAtTab(runs()) : null);
	const editor = () =>
		props.block.slotId && !props.block.section ? props.editor : undefined;
	const lead = () => dominantRun(props.block.runs);
	return (
		<div
			data-slot-id={props.block.slotId || undefined}
			data-section={props.block.section || undefined}
			class="relative whitespace-pre-wrap [font-kerning:normal] [font-variant-ligatures:none] [tab-size:36pt]"
			style={blockStyle(props.block, props.draw, spacer())}
		>
			<Show when={editor()}>
				{(ed) => (
					<>
						<Show when={ed().active === props.block.slotId}>
							<span
								aria-hidden="true"
								class="absolute -left-2 inset-y-0 w-0.5 rounded-full bg-primary"
							/>
						</Show>
						<Show when={ed().showCounts && props.fit}>
							{(fit) => (
								<span
									aria-hidden="true"
									data-testid="line-count"
									title={`${fit().lines} lines, last line ${Math.round(fit().lastLineFill * 100)}% full`}
									class={cn(
										"pointer-events-none absolute top-0 text-right font-mono text-2xs tabular-nums",
										fit().lines > 1 && fit().lastLineFill < SPARSE_LAST_LINE
											? "text-status-interview"
											: "text-faint",
									)}
									style={{
										left: `${-props.page.marginLeft - props.block.indentStart}pt`,
										width: `${Math.max(props.page.marginLeft - 14, 0)}pt`,
									}}
								>
									{fit().lines}
								</span>
							)}
						</Show>
					</>
				)}
			</Show>
			<Show when={props.block.bullet}>
				{(bullet) => (
					<span
						aria-hidden="true"
						class="absolute"
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
						<Show when={split()} fallback={<Runs runs={runs()} />}>
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
					}
				>
					{(ed) => (
						<EditableLine
							slotId={props.block.slotId}
							editor={ed()}
							runStyle={lead() ? runStyle(lead() as LayoutRun) : {}}
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
			bodyLinePt: (bodyLinePx || 15 * PX_PER_PT) / PX_PER_PT,
			fits: next,
		});
	};
	const remeasure = () => queueMicrotask(measure);

	const measureLines = (slotId: string, text: string) => {
		const el = lines.get(slotId);
		const block = el?.closest<HTMLElement>("[data-slot-id]");
		if (!block?.parentElement) return 1;
		const clone = block.cloneNode(true) as HTMLElement;
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
		const editor = props.editor;
		if (editor)
			for (const b of props.layout.blocks) if (b.slotId) editor.text(b.slotId);
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
