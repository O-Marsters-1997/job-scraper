import { useQueryClient } from "@tanstack/solid-query";
import { type Accessor, createSignal, onCleanup, onMount } from "solid-js";
import { createStore, unwrap } from "solid-js/store";
import { keys } from "@/api/keys";
import { saveDraftSlots } from "@/api/tailoring";
import { blockText } from "@/lib/docLayout";
import { markNovel } from "@/lib/markNovel";
import { createSaveLoop, type SaveStatus } from "@/lib/saveLoop";
import { reviewFindings, skillGaps } from "@/lib/tailoring";
import type { Draft, DraftLayout, Segment } from "@/types/tailoring";

export const PROFILE_SLOT = "profile";
export const SKILLS_CARD = "skills";

const SAVE_DELAY_MS = 1500;
const PAGE_COUNT = /is (\d+) pages?/;

const oneLine = (text: string) => text.replace(/\s*\n\s*/g, " ");

function resolvedKey(id: string) {
	return `draft.resolved.${id}`;
}

function readResolved(id: string): Record<string, boolean> {
	try {
		return JSON.parse(localStorage.getItem(resolvedKey(id)) ?? "{}");
	} catch {
		return {};
	}
}

export function createDraftEditor(
	draft: Accessor<Draft>,
	layout: DraftLayout | null,
) {
	const queryClient = useQueryClient();
	const id = draft().id;
	const slotIds = (layout?.blocks ?? [])
		.filter((b) => b.slotId && !b.section)
		.map((b) => b.slotId);
	const original: Record<string, string> = {};
	for (const b of layout?.blocks ?? [])
		if (b.slotId && !b.section) original[b.slotId] = blockText(b);

	const [texts, setTexts] = createStore<Record<string, string>>({
		...original,
	});
	const [saved, setSaved] = createStore<Record<string, string>>({
		...original,
	});
	const [status, setStatus] = createSignal<SaveStatus>("saved");
	const [googleSaved, setGoogleSaved] = createSignal(false);
	const [resolved, setResolvedMap] = createSignal(readResolved(id));

	const positions = () => draft().provenance?.positions ?? [];
	const bulletOf = (slotId: string) =>
		positions()
			.flatMap((p) => p.bullets)
			.find((b) => b.slotId === slotId);
	const labels = () => {
		const out: Record<string, string> = { [PROFILE_SLOT]: "Profile" };
		for (const p of positions())
			p.bullets.forEach((b, i) => {
				out[b.slotId] = `${p.title} bullet ${i + 1}`;
			});
		return out;
	};
	const label = (slotId: string) => labels()[slotId] ?? "Line";

	const serverSegments = (slotId: string): Segment[] | undefined =>
		slotId === PROFILE_SLOT
			? (draft().provenance?.profile?.segments ?? undefined)
			: bulletOf(slotId)?.segments;
	const sources = (slotId: string): string[] => {
		if (slotId !== PROFILE_SLOT)
			return bulletOf(slotId)?.achievements.map((a) => a.text) ?? [];
		return [
			...positions().flatMap((p) =>
				p.bullets.flatMap((b) => b.achievements.map((a) => a.text)),
			),
			...(serverSegments(PROFILE_SLOT) ?? [])
				.filter((s) => !s.novel)
				.map((s) => s.text),
		];
	};
	const text = (slotId: string) => texts[slotId] ?? "";
	const segments = (slotId: string): Segment[] => {
		const server = serverSegments(slotId);
		if (server && server.map((s) => s.text).join("") === text(slotId))
			return server;
		return markNovel(text(slotId), sources(slotId));
	};

	const send = async () => {
		const sent = { ...unwrap(texts) };
		const slots = slotIds
			.map((slotId) => ({ slotId, text: (sent[slotId] ?? "").trim() }))
			.filter((s) => s.text !== "");
		const result = await saveDraftSlots(id, slots);
		setSaved(Object.fromEntries(slots.map((s) => [s.slotId, sent[s.slotId]])));
		setGoogleSaved(true);
		queryClient.setQueryData(keys.tailoring.draft(id), result);
	};
	const loop = createSaveLoop({
		delay: SAVE_DELAY_MS,
		send,
		onStatus: setStatus,
	});

	onMount(() => {
		const warn = (e: BeforeUnloadEvent) => {
			if (loop.pending()) e.preventDefault();
		};
		window.addEventListener("beforeunload", warn);
		onCleanup(() => window.removeEventListener("beforeunload", warn));
	});
	onCleanup(() => {
		void loop.flush();
	});

	const setText = (slotId: string, next: string) => {
		const clean = oneLine(next);
		if (clean === text(slotId)) return;
		setTexts(slotId, clean);
		loop.schedule();
	};

	const dirty = () => slotIds.some((s) => text(s) !== saved[s]);
	const pageCountFinding = () =>
		draft().findings.find((f) => f.check === "page_count");
	const googlePages = (): number | undefined => {
		const finding = pageCountFinding();
		if (dirty() || !finding) return undefined;
		const n = Number(PAGE_COUNT.exec(finding.message)?.[1]);
		return n > 1 ? n : 2;
	};

	const googleAgrees = () => googleSaved() && !dirty() && !pageCountFinding();

	const findingsOf = (slotId: string) =>
		slotId === PROFILE_SLOT
			? []
			: reviewFindings(draft().findings).filter((f) => f.slotId === slotId);
	const gaps = () => skillGaps(draft().findings);
	const hasSkills = () =>
		(layout?.blocks ?? []).some((b) => b.section === "skills");
	const cardKeys = (hasCard: (slotId: string) => boolean) => [
		...slotIds.filter((slotId) => hasCard(slotId) || findingsOf(slotId).length),
		...(gaps().length && hasSkills() ? [SKILLS_CARD] : []),
	];

	const setResolved = (key: string, value: boolean) => {
		const next = { ...resolved(), [key]: value };
		setResolvedMap(next);
		try {
			localStorage.setItem(resolvedKey(id), JSON.stringify(next));
		} catch {}
	};

	return {
		slotIds,
		label,
		text,
		original: (slotId: string) => original[slotId] ?? "",
		setText,
		undo: (slotId: string) => setText(slotId, original[slotId] ?? ""),
		edited: (slotId: string) => text(slotId) !== original[slotId],
		segments,
		findingsFor: findingsOf,
		gaps,
		cardKeys,
		isResolved: (key: string) => resolved()[key] === true,
		setResolved,
		status,
		retry: loop.retry,
		flush: loop.flush,
		googlePages,
		googleAgrees,
	};
}

export type DraftEditorState = ReturnType<typeof createDraftEditor>;
