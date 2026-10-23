import { Link } from "@tanstack/solid-router";
import {
	createEffect,
	createMemo,
	createSignal,
	Match,
	onCleanup,
	onMount,
	Show,
	Switch,
	untrack,
} from "solid-js";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { isSparse, maxCharsForFewerLines } from "@/lib/docLayout";
import { loadedData } from "@/lib/queryData";
import { cn } from "@/lib/utils";
import { wordDiff } from "@/lib/wordDiff";
import type { Draft, DraftLayout } from "@/types/tailoring";
import { createDraftEditor } from "../../hooks/useDraftEditor";
import { useExperience } from "../../hooks/useExperience";
import { useJob } from "../../hooks/useJobs";
import { createKeepFlow } from "../../hooks/useKeepFlow";
import { createRolePicks } from "../../hooks/useRolePicks";
import { createSuggestions } from "../../hooks/useSuggestions";
import {
	useSuggestions as useAchievementBank,
	useDraftLean,
} from "../../hooks/useTailoring";
import { AchievementsDrawer } from "./AchievementsDrawer";
import { ChangesDiff } from "./ChangesDiff";
import {
	DEFAULT_BODY_LINE_PT,
	DocPage,
	type PageEditor,
	type PageMetrics,
	PX_PER_PT,
} from "./DocPage";
import { PageMeter, SaveIndicator } from "./DraftMeters";
import { KeepControls, KeepNotices } from "./KeepFlow";
import { MarginCards } from "./MarginCards";
import { PrintPreview } from "./PrintPreview";
import { RolesDrawer } from "./RolesDrawer";
import { SkillsPanel } from "./SkillsPanel";
import { WandMenu } from "./WandMenu";

const NO_METRICS: PageMetrics = {
	contentPt: 0,
	availablePt: 1,
	bodyLinePt: DEFAULT_BODY_LINE_PT,
	fits: {},
};

const RAIL_PX = 320;
const RAIL_GAP_PX = 48;
const DESKTOP_PAD_PX = 64;
const NARROW_PAD_PX = 32;

function watchSize(el: HTMLElement, onResize: () => void) {
	const ro = new ResizeObserver(onResize);
	ro.observe(el);
	onCleanup(() => ro.disconnect());
}

function LayoutUnavailable(props: { reason: string | undefined }) {
	return (
		<div class="mx-auto max-w-3xl px-6 py-10">
			<div
				role="alert"
				class="rounded-xl border border-border bg-surface p-6 text-sm"
			>
				<p class="font-medium text-foreground">
					This CV cannot be shown as a page here.
				</p>
				<p class="mt-1 text-muted">
					{props.reason ?? "Its layout is not available."} Use Print preview to
					read Google's own export, or open the Doc in Google Docs.
				</p>
			</div>
		</div>
	);
}

export function DraftEditor(props: {
	draft: Draft;
	layout: DraftLayout | null;
	layoutError?: string | undefined;
}) {
	const ed = createDraftEditor(
		() => props.draft,
		() => props.layout,
	);
	const flow = createKeepFlow(() => props.draft, ed.flush);
	const job = useJob(() => props.draft.jobId);

	const [active, setActive] = createSignal<string>();
	const [metrics, setMetrics] = createSignal(NO_METRICS);
	const [showCounts, setShowCounts] = createSignal(true);
	const [showChanges, setShowChanges] = createSignal(false);
	const [preview, setPreview] = createSignal(false);
	const [skillsOpen, setSkillsOpen] = createSignal(false);
	const [stage, setStage] = createSignal<HTMLDivElement>();
	const [askSlot, setAskSlot] = createSignal<string>();
	const [mainWidth, setMainWidth] = createSignal(0);
	const [innerHeight, setInnerHeight] = createSignal(0);
	const [wandTop, setWandTop] = createSignal(0);
	let measureLines: (slotId: string, text: string) => number = () => 1;

	const editable = () =>
		props.draft.status === "ready" && props.draft.outcome === null;
	const changes = () => {
		const { base, content } = props.draft;
		return base && content ? { base, content } : null;
	};

	const leanQuery = useDraftLean(
		() => props.draft.id,
		() => props.draft.status === "ready",
	);
	const leanBySlot = createMemo(
		() => new Map(leanQuery.data?.map((b) => [b.slotId, b])),
	);

	const pagePx = () => (props.layout?.page.width ?? 0) * PX_PER_PT;
	const narrow = () =>
		mainWidth() > 0 &&
		mainWidth() < pagePx() + RAIL_PX + RAIL_GAP_PX + DESKTOP_PAD_PX;
	const scale = () =>
		narrow() ? Math.min(1, (mainWidth() - NARROW_PAD_PX) / pagePx()) : 1;

	const suggestions = createSuggestions({
		// eslint-disable-next-line solid/reactivity -- pedantic: the Draft id is fixed for this editor
		draftId: props.draft.id,
		text: ed.text,
		maxChars: (slotId) => {
			const fit = metrics().fits[slotId];
			const text = ed.text(slotId);
			return fit
				? maxCharsForFewerLines(text, fit.lines, fit.lastLineFill)
				: text.length;
		},
		apply: ed.setText,
	});
	const needsCard = (slotId: string) =>
		ed.edited(slotId) ||
		!!suggestions.get(slotId) ||
		isSparse(metrics().fits[slotId]);
	const cardKeys = createMemo(() => ed.cardKeys(needsCard));

	const activeSlot = () => {
		const key = active();
		return key && ed.slotIds().includes(key) ? key : undefined;
	};
	const wandSlot = () => {
		const key = activeSlot();
		return key && editable() && !suggestions.get(key) ? key : undefined;
	};
	const askOpen = () => askSlot() !== undefined && askSlot() === activeSlot();
	const setAskOpen = (open: boolean) =>
		setAskSlot(open ? activeSlot() : undefined);
	const activate = (key: string | undefined) => {
		if (key !== active()) setAskSlot(undefined);
		setActive(key);
	};
	const focusLine = (slotId: string, atEnd = false) => {
		const root = stage();
		queueMicrotask(() => {
			const line = root?.querySelector<HTMLElement>(
				`[data-slot-id="${slotId}"] [role="textbox"]`,
			);
			line?.focus({ preventScroll: true });
			if (line && atEnd) {
				const caret = document.createRange();
				caret.selectNodeContents(line);
				caret.collapse(false);
				const sel = window.getSelection();
				sel?.removeAllRanges();
				sel?.addRange(caret);
			}
		});
	};
	const fitSlot = (slotId: string) => {
		activate(slotId);
		void suggestions.ask(slotId, "fit");
	};
	const openPreview = async () => {
		if (await ed.flush()) setPreview(true);
	};

	const [rolePanel, setRolePanel] = createSignal<string>();
	const [selectedPick, setSelectedPick] = createSignal<string>();
	const [bankWanted, setBankWanted] = createSignal(false);
	const bank = useAchievementBank(
		() => props.draft.jobId,
		() =>
			bankWanted()
				? { docId: props.draft.baseDocId, tabId: props.draft.baseTabId }
				: undefined,
	);
	const rolePicks = createRolePicks({
		draft: () => props.draft,
		editor: ed,
		bank: () => bank.data ?? [],
		positions: () => rolePositions(),
	});
	const pageLines = (pt: number) =>
		Math.round(pt / (metrics().bodyLinePt || DEFAULT_BODY_LINE_PT));
	const [rolesOpen, setRolesOpen] = createSignal(false);
	const [selectedRole, setSelectedRole] = createSignal<string>();
	const experience = useExperience();
	const rolePositions = () =>
		[...(loadedData(experience) ?? [])].sort((a, b) =>
			(b.startDate ?? "9999").localeCompare(a.startDate ?? "9999"),
		);
	const closeRoles = () => {
		setSelectedRole(undefined);
		setRolesOpen(false);
	};
	const openRoles = () => {
		closeRole();
		setBankWanted(true);
		setRolesOpen(true);
	};
	const openRole = (positionId: string) => {
		closeRoles();
		setSelectedPick(undefined);
		setBankWanted(true);
		setRolePanel(positionId);
	};
	const closeRole = () => {
		setSelectedPick(undefined);
		setRolePanel(undefined);
		activate(undefined);
	};
	createEffect(() => {
		const positionId = rolePanel();
		if (!positionId) return;
		const id = selectedPick();
		const at = id ? rolePicks.picksFor(positionId).indexOf(id) : -1;
		const slot = at >= 0 ? rolePicks.roleSlots(positionId)[at] : undefined;
		untrack(() => activate(slot));
	});

	const pageEditor: PageEditor = {
		get editable() {
			return editable();
		},
		get showCounts() {
			return showCounts();
		},
		get active() {
			return active();
		},
		text: ed.text,
		label: ed.label,
		lean: (slotId) => leanBySlot().get(slotId),
		diffFor: (slotId) => {
			const sug = suggestions.get(slotId);
			if (!sug || sug.error !== undefined) return null;
			return sug.done
				? wordDiff(sug.before, sug.text)
				: [
						{ op: "del", text: `${sug.before} ` },
						{ op: "add", text: sug.text },
					];
		},
		onInput: ed.setText,
		onFocus: activate,
		onBlur: () => void ed.flush(),
		onEnter: (slotId) => {
			const added = ed.addBullet(slotId);
			if (added) focusLine(added);
		},
		onBackspaceEmpty: (slotId) => {
			const previous = ed.removeBullet(slotId);
			if (previous === undefined) return false;
			focusLine(previous, true);
			return true;
		},
		get onEditSkills() {
			return editable() && props.draft.skillsEditable
				? () => void setSkillsOpen(true)
				: undefined;
		},
		roleFor: (block) => rolePicks.headings().get(block),
		get onEditRole() {
			return editable() ? openRole : undefined;
		},
	};

	createEffect(() => {
		const key = activeSlot();
		const root = stage();
		metrics();
		scale();
		void suggestions.get(key ?? "")?.done;
		const line = key && root?.querySelector(`[data-slot-id="${key}"]`);
		if (line && root)
			setWandTop(
				line.getBoundingClientRect().top - root.getBoundingClientRect().top,
			);
	});
	onMount(() => {
		const onKey = (e: KeyboardEvent) => {
			const key = active();
			if (!key) return;
			const sug = suggestions.get(key);
			const mod = e.metaKey || e.ctrlKey;
			if (e.key === "Escape" && sug) {
				suggestions.reject(key);
				focusLine(key);
			} else if (mod && e.key === "k" && wandSlot()) {
				e.preventDefault();
				setAskOpen(true);
			} else if (mod && e.key === "Enter" && sug?.done) {
				e.preventDefault();
				suggestions.accept(key);
				focusLine(key);
			}
		};
		window.addEventListener("keydown", onKey);
		onCleanup(() => window.removeEventListener("keydown", onKey));
	});

	const marginCards = (narrowRail: boolean) => (
		<MarginCards
			narrow={narrowRail}
			editor={ed}
			keys={cardKeys()}
			suggestions={suggestions}
			measureLines={(id, text) => measureLines(id, text)}
			stage={stage()}
			fits={metrics().fits}
			active={active()}
			editable={editable()}
			onActive={activate}
			onFit={editable() ? fitSlot : undefined}
		/>
	);

	return (
		<div class="fixed inset-0 z-50 flex flex-col bg-background">
			<header class="flex min-h-14 shrink-0 flex-wrap items-center gap-x-3 gap-y-1 border-b border-border bg-surface px-4 py-1.5">
				<Link
					to="/jobs/$id"
					params={{ id: props.draft.jobId }}
					aria-label="Back to job"
					class="grid size-8 place-items-center rounded-md text-muted transition-colors hover:bg-accent-subtle hover:text-foreground"
				>
					<Icon name="chevronLeft" size={16} />
				</Link>
				<div class="w-32 shrink-0">
					<h1 class="truncate text-sm font-semibold text-foreground">
						Draft CV
					</h1>
					<p class="truncate text-xs text-faint">
						{job.data?.Title ?? "Loading job…"}
					</p>
				</div>
				<Show when={editable()}>
					<SaveIndicator status={ed.status()} onRetry={ed.retry} />
				</Show>
				<div class="order-last flex w-full flex-wrap items-center justify-center gap-x-4 gap-y-1 lg:order-none lg:w-auto lg:flex-1">
					<Show when={props.layout}>
						<PageMeter
							metrics={metrics()}
							googlePages={ed.googlePages()}
							googleAgrees={ed.googleAgrees()}
						/>
						<span class="h-4 w-px bg-border" />
						<span
							class="font-mono text-xs whitespace-nowrap text-faint tabular-nums"
							data-testid="resolved-count"
						>
							{cardKeys().filter((k) => ed.isResolved(k)).length}/
							{cardKeys().length} resolved
						</span>
						<Button
							variant="ghost"
							size="sm"
							aria-pressed={showCounts()}
							onClick={() => setShowCounts((v) => !v)}
						>
							Line counts
						</Button>
					</Show>
				</div>
				<div class="ml-auto flex flex-wrap items-center justify-end gap-1.5 lg:ml-0">
					<Show when={changes()}>
						<Button
							variant="ghost"
							size="sm"
							aria-pressed={showChanges()}
							onClick={() => setShowChanges((v) => !v)}
						>
							Changes
						</Button>
					</Show>
					<Show when={editable() && props.layout}>
						<Button
							variant="ghost"
							size="sm"
							aria-label="Roles"
							onClick={openRoles}
						>
							<Icon name="briefcase" size={14} />
							<span class="hidden 2xl:inline">Roles</span>
						</Button>
					</Show>
					<Button
						variant="ghost"
						size="sm"
						aria-label="Print preview"
						onClick={openPreview}
					>
						<Icon name="fileText" size={14} />
						<span class="hidden 2xl:inline">Print preview</span>
					</Button>
					<Show when={props.draft.draftDocUrl}>
						{(url) => (
							<Button
								as="a"
								href={url()}
								target="_blank"
								rel="noopener noreferrer"
								variant="ghost"
								size="sm"
								aria-label="Open in Google Docs"
							>
								<Icon name="externalLink" size={14} />
								<span class="hidden 2xl:inline">Google Docs</span>
							</Button>
						)}
					</Show>
					<span aria-hidden="true" class="mx-1 h-5 w-px bg-border" />
					<KeepControls draft={props.draft} flow={flow} />
				</div>
			</header>

			<KeepNotices draft={props.draft} editable={editable()} flow={flow} />

			<main
				ref={(el) => watchSize(el, () => setMainWidth(el.clientWidth))}
				class={cn(
					"scroll-slim flex-1 overflow-auto bg-border/45",
					(rolePanel() || rolesOpen()) && "pr-[30rem]",
				)}
				onMouseDown={(e) => {
					if (e.target === e.currentTarget) activate(undefined);
				}}
			>
				<Switch fallback={<LayoutUnavailable reason={props.layoutError} />}>
					<Match when={showChanges() && changes()}>
						{(c) => (
							<div class="mx-auto max-w-3xl px-6 py-10">
								<section
									aria-label="Changes from your base CV"
									class="rounded-xl border border-border bg-surface p-6"
								>
									<h2 class="mb-3 text-sm font-semibold">
										Changes from your base CV
									</h2>
									<ChangesDiff
										base={c().base}
										content={c().content}
										provenance={props.draft.provenance}
										onUndo={editable() ? ed.setText : undefined}
									/>
								</section>
							</div>
						)}
					</Match>
					<Match when={props.layout}>
						{(layout) => (
							<div
								class={cn(
									"mx-auto flex w-max items-start",
									narrow() ? "px-4 pt-4 pb-40" : "gap-12 px-8 pt-16 pb-40",
								)}
							>
								<div
									ref={setStage}
									class="relative"
									style={
										scale() < 1
											? {
													width: `${pagePx() * scale()}px`,
													height: `${innerHeight() * scale()}px`,
												}
											: {}
									}
								>
									<div
										ref={(el) =>
											watchSize(el, () => setInnerHeight(el.offsetHeight))
										}
										style={
											scale() < 1
												? {
														width: `${pagePx()}px`,
														transform: `scale(${scale()})`,
														"transform-origin": "top left",
													}
												: {}
										}
									>
										<DocPage
											layout={{ ...layout(), blocks: ed.blocks() }}
											editor={pageEditor}
											onMetrics={setMetrics}
											onMeasurer={(fn) => {
												measureLines = fn;
											}}
										/>
									</div>
									<Show when={wandSlot()}>
										{(key) => (
											<WandMenu
												top={wandTop()}
												canFit={(metrics().fits[key()]?.lines ?? 1) > 1}
												askOpen={askOpen()}
												onAskOpen={setAskOpen}
												onAction={(action, prompt) =>
													void suggestions.ask(key(), action, prompt)
												}
											/>
										)}
									</Show>
									<Show when={narrow()}>{marginCards(true)}</Show>
								</div>
								<Show when={!narrow() && !rolePanel() && !rolesOpen()}>
									{marginCards(false)}
								</Show>
							</div>
						)}
					</Match>
				</Switch>
			</main>
			<div class="sr-only" aria-live="polite">
				{suggestions.announcement()}
			</div>
			<Show when={rolesOpen()}>
				<RolesDrawer
					positions={rolePositions()}
					picked={rolePicks.order()}
					usedLines={pageLines(metrics().contentPt)}
					budgetLines={pageLines(metrics().availablePt)}
					selected={selectedRole()}
					onSelect={setSelectedRole}
					onToggle={rolePicks.toggleRole}
					onChooseAchievements={openRole}
					onMove={rolePicks.moveRole}
					onClose={closeRoles}
				/>
			</Show>
			<Show when={rolePanel()}>
				{(positionId) => {
					const role = () => rolePicks.role(positionId());
					return (
						<AchievementsDrawer
							title={`${role()?.title ?? ""}, ${role()?.employer ?? ""}`}
							bank={rolePicks.bankFor(positionId())}
							emptyText="No Achievements found for this role."
							loading={bank.isPending}
							picked={rolePicks.picksFor(positionId())}
							usedLines={pageLines(metrics().contentPt)}
							budgetLines={pageLines(metrics().availablePt)}
							selected={selectedPick()}
							onSelect={setSelectedPick}
							onToggle={(id) => rolePicks.toggle(positionId(), id)}
							onMove={(id, place) => rolePicks.move(positionId(), id, place)}
							onClear={() => rolePicks.clear(positionId())}
							onClose={closeRole}
						/>
					);
				}}
			</Show>
			<Show when={props.draft.skillsEditable}>
				<SkillsPanel
					draft={props.draft}
					open={skillsOpen()}
					onClose={() => setSkillsOpen(false)}
					flush={ed.flush}
				/>
			</Show>
			<Show when={preview()}>
				<PrintPreview
					draftId={props.draft.id}
					onClose={() => setPreview(false)}
				/>
			</Show>
		</div>
	);
}
