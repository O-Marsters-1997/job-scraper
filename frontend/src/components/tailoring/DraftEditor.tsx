import { Link } from "@tanstack/solid-router";
import {
	createEffect,
	createMemo,
	createSignal,
	For,
	Match,
	onCleanup,
	onMount,
	Show,
	Switch,
	untrack,
} from "solid-js";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { inlineMarks } from "@/lib/bulletDiff";
import { isSparse, maxCharsForFewerLines } from "@/lib/docLayout";
import { htmlToText } from "@/lib/htmlText";
import { loadedData } from "@/lib/queryData";
import { roleBudget } from "@/lib/tailoring";
import { cn } from "@/lib/utils";
import { wordDiff } from "@/lib/wordDiff";
import type { Draft, DraftLayout } from "@/types/tailoring";
import { createDraftEditor, PROFILE_SLOT } from "../../hooks/useDraftEditor";
import { useExperience } from "../../hooks/useExperience";
import { useJobComments } from "../../hooks/useJobComments";
import { useJobNote } from "../../hooks/useJobNotes";
import { useJob } from "../../hooks/useJobs";
import { createKeepFlow } from "../../hooks/useKeepFlow";
import { createRolePicks } from "../../hooks/useRolePicks";
import { createSuggestions } from "../../hooks/useSuggestions";
import {
	useSuggestions as useAchievementBank,
	useDraftInsights,
	useDraftLean,
	useDraftRoles,
	useExplainDraft,
	useRedraftProfile,
} from "../../hooks/useTailoring";
import { AchievementsDrawer } from "./AchievementsDrawer";
import {
	DEFAULT_BODY_LINE_PT,
	DocPage,
	type PageDiff,
	type PageEditor,
	type PageMetrics,
	PX_PER_PT,
} from "./DocPage";
import { PageMeter, SaveIndicator } from "./DraftMeters";
import { Insights } from "./Insights";
import type { JobTextBlock } from "./JobText";
import { KeepControls, KeepNotices } from "./KeepFlow";
import { MarginCards } from "./MarginCards";
import { PrintPreview } from "./PrintPreview";
import { ProfileCard } from "./ProfileCard";
import { RolesDrawer } from "./RolesDrawer";
import { SkillsPanel } from "./SkillsPanel";
import { UnlinkedRoles } from "./UnlinkedRoles";
import { RedraftWand, WandMenu } from "./WandMenu";

const NO_METRICS: PageMetrics = {
	contentPt: 0,
	availablePt: 1,
	bodyLinePt: DEFAULT_BODY_LINE_PT,
	fits: {},
};

type View = "document" | "insights";

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
	const [hovered, setHovered] = createSignal<string>();
	const hover = (key: string, over: boolean) =>
		setHovered((cur) => (over ? key : cur === key ? undefined : cur));
	const [metrics, setMetrics] = createSignal(NO_METRICS);
	const [showCounts, setShowCounts] = createSignal(true);
	const [view, setView] = createSignal<View>("document");
	const [showResolved, setShowResolved] = createSignal(false);
	const [showDiff, setShowDiff] = createSignal(false);
	const [preview, setPreview] = createSignal(false);
	const [skillsOpen, setSkillsOpen] = createSignal(false);
	const [stage, setStage] = createSignal<HTMLDivElement>();
	const [askSlot, setAskSlot] = createSignal<string>();
	const [mainWidth, setMainWidth] = createSignal(0);
	const [innerHeight, setInnerHeight] = createSignal(0);
	const [wandTop, setWandTop] = createSignal(0);
	let measureLines: (slotId: string, text: string) => number = () => 1;

	const editable = () =>
		props.draft.status === "ready" && props.draft.outcome !== "discarded";
	const keepable = () => editable() && props.draft.outcome === null;
	const changes = () => {
		const { base, content } = props.draft;
		return base && content ? { base, content } : null;
	};
	const pageDiff = (): PageDiff | undefined => {
		const c = changes();
		if (!showDiff() || !c) return undefined;
		const added = new Set<string>();
		const removedAfter: Record<string, string[]> = {};
		for (const p of props.draft.provenance?.positions ?? []) {
			const slots = p.bullets.map((b) => b.slotId);
			const base = c.base.positions.find((x) => x.positionId === p.positionId);
			const marks = inlineMarks(
				base?.bullets.map((b) => b.text) ?? [],
				slots.map(ed.text),
			);
			slots.forEach((s, i) => {
				if (marks.added[i]) added.add(s);
			});
			const last = slots.at(-1);
			if (last && marks.removed.length > 0) removedAfter[last] = marks.removed;
		}
		return { added, removedAfter };
	};

	const leanQuery = useDraftLean(
		() => props.draft.id,
		() => props.draft.status === "ready",
	);
	const insightsQuery = useDraftInsights(
		() => props.draft.id,
		() => props.draft.status === "ready",
	);
	const explain = useExplainDraft(() => props.draft.id);
	const note = useJobNote(() => props.draft.jobId);
	const comments = useJobComments(() => props.draft.jobId);
	const jobText = () => {
		const blocks: JobTextBlock[] = [
			{
				label: "Job description",
				source: "jd",
				text: htmlToText(loadedData(job)?.Description ?? ""),
			},
			{
				label: "Your note",
				source: "note",
				text: loadedData(note)?.Body ?? "",
			},
			...(loadedData(comments) ?? []).map(
				(c): JobTextBlock => ({
					label: "Comment",
					source: "comment",
					text: c.Body,
				}),
			),
		];
		return blocks.filter((b) => b.text !== "");
	};
	const leanBySlot = createMemo(
		() => new Map(loadedData(leanQuery)?.map((b) => [b.slotId, b])),
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
	const cardKeys = createMemo(() =>
		ed.cardKeys(needsCard).filter((k) => k !== PROFILE_SLOT),
	);

	const redraft = useRedraftProfile(() => props.draft.id);
	const [lastRedraft, setLastRedraft] = createSignal<{
		from: string;
		to: string;
	}>();
	const [profileTop, setProfileTop] = createSignal(0);
	const baseProfile = () => props.draft.base?.profile ?? undefined;
	const runRedraft = async (steer?: string) => {
		setLastRedraft(undefined);
		await ed.flush();
		const from = ed.text(PROFILE_SLOT);
		redraft.mutate(steer ? { steer } : {}, {
			onSuccess: (saved) => {
				const next = saved.content?.profile;
				if (next == null || next === from) return;
				ed.setText(PROFILE_SLOT, next);
				setLastRedraft({ from, to: ed.text(PROFILE_SLOT) });
			},
		});
	};
	const revertProfile = () => {
		const last = lastRedraft();
		const undo = last && ed.text(PROFILE_SLOT) === last.to;
		const to = undo ? last.from : baseProfile();
		setLastRedraft(undefined);
		redraft.reset();
		if (to === undefined) return;
		ed.setText(PROFILE_SLOT, to);
		void ed.flush();
	};

	const activeSlot = () => {
		const key = active();
		return key && ed.slotIds().includes(key) ? key : undefined;
	};
	const wandSlot = () => {
		const key = activeSlot();
		return key && key !== PROFILE_SLOT && editable() && !suggestions.get(key)
			? key
			: undefined;
	};
	const profileWand = () =>
		activeSlot() === PROFILE_SLOT && editable() && !!baseProfile();
	const focusSteer = () =>
		document
			.querySelector<HTMLInputElement>(
				'[data-testid="profile-card"] input[aria-label="Steer the redraft"]',
			)
			?.focus();
	const tailorQuote = (slotId: string) =>
		loadedData(insightsQuery)?.lines.find(
			(l) => l.slotId === slotId && l.anchored,
		);
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
	const editInDocument = (slotId: string) => {
		setView("document");
		activate(slotId);
		focusLine(slotId, true);
	};
	const tabs = (): { id: View; label: string }[] => [
		{ id: "document", label: "Document" },
		...(loadedData(leanQuery)
			? [{ id: "insights" as const, label: "Insights" }]
			: []),
	];
	const current = () =>
		tabs().some((t) => t.id === view()) ? view() : "document";
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
		bank: () => loadedData(bank) ?? [],
		positions: () => rolePositions(),
	});
	const rolesQuery = useDraftRoles(
		() => props.draft.id,
		() => props.draft.status === "ready",
	);
	const budgetOf = (positionId: string) => {
		const role = loadedData(rolesQuery)?.roles.find(
			(r) => r.positionId === positionId,
		);
		return role && roleBudget(role);
	};
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
	const [roleFromList, setRoleFromList] = createSignal(false);
	const openRole = (positionId: string) => {
		setRoleFromList(rolesOpen());
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
		hasCard: (slotId) => cardKeys().includes(slotId),
		onHover: hover,
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
		onDismissSuggestion: suggestions.reject,
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
		isPlaceholder: rolePicks.isPlaceholder,
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
	createEffect(() => {
		const root = stage();
		metrics();
		scale();
		ed.text(PROFILE_SLOT);
		const rect = root
			?.querySelector(`[data-slot-id="${PROFILE_SLOT}"]`)
			?.getBoundingClientRect();
		if (rect && root) {
			const stageTop = root.getBoundingClientRect().top;
			setProfileTop(
				narrow() ? rect.bottom - stageTop + 6 : rect.top - stageTop,
			);
		}
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
			showResolved={showResolved()}
			suggestions={suggestions}
			measureLines={(id, text) => measureLines(id, text)}
			stage={stage()}
			fits={metrics().fits}
			active={active()}
			hovered={hovered()}
			onHover={hover}
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
						{loadedData(job)?.Title ?? "Loading job…"}
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
							aria-pressed={showResolved()}
							onClick={() => setShowResolved((v) => !v)}
						>
							Show resolved
						</Button>
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
					<nav aria-label="View" class="flex gap-0.5">
						<For each={tabs()}>
							{(tab) => (
								<Button
									variant="ghost"
									size="sm"
									aria-pressed={current() === tab.id}
									onClick={() => setView(tab.id)}
								>
									{tab.label}
								</Button>
							)}
						</For>
					</nav>
					<Show when={editable() && props.layout}>
						<Button
							variant="ghost"
							size="sm"
							aria-pressed={showDiff()}
							onClick={() => setShowDiff((v) => !v)}
						>
							Diff
						</Button>
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

			<KeepNotices draft={props.draft} editable={keepable()} flow={flow} />
			<Show when={editable()}>
				<UnlinkedRoles draft={props.draft} />
			</Show>

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
					<Match when={current() === "insights" && loadedData(leanQuery)}>
						{(lean) => (
							<div class="mx-auto max-w-6xl px-6 py-10">
								<Insights
									lean={lean()}
									jobText={jobText()}
									insights={loadedData(insightsQuery)}
									explaining={explain.isPending}
									explainError={explain.error?.message}
									text={ed.text}
									onEdit={editInDocument}
									onExplain={() => explain.mutate()}
								/>
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
											diff={pageDiff()}
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
												tailor={tailorQuote(key())}
												askOpen={askOpen()}
												onAskOpen={setAskOpen}
												onAction={(action, prompt, quote) =>
													void suggestions.ask(key(), action, prompt, quote)
												}
											/>
										)}
									</Show>
									<Show when={profileWand()}>
										<RedraftWand top={wandTop()} onRedraft={focusSteer} />
									</Show>
									<Show
										when={
											editable() &&
											!!baseProfile() &&
											(narrow()
												? active() === PROFILE_SLOT
												: !rolePanel() && !rolesOpen())
										}
									>
										<div
											class={cn(
												"absolute top-0",
												narrow()
													? "inset-x-2 z-20"
													: "left-[calc(100%+3rem)] z-20 w-80",
											)}
										>
											<ProfileCard
												base={baseProfile() ?? ""}
												text={ed.text(PROFILE_SLOT)}
												top={profileTop()}
												pending={redraft.isPending}
												{...(redraft.error && {
													error: redraft.error.message,
												})}
												onRedraft={(steer) => void runRedraft(steer)}
												onRevert={revertProfile}
											/>
										</div>
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
					budgetOf={budgetOf}
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
							roleBudget={budgetOf(positionId())}
							selected={selectedPick()}
							onSelect={setSelectedPick}
							onToggle={(id) => rolePicks.toggle(positionId(), id)}
							onMove={(id, place) => rolePicks.move(positionId(), id, place)}
							carried={rolePicks.carriedFor(positionId())}
							onEmpty={() => rolePicks.empty(positionId())}
							onReset={() => rolePicks.reset(positionId())}
							onBackToRoles={roleFromList() ? openRoles : undefined}
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
