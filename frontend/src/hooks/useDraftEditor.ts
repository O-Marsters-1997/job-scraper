import { useQueryClient } from "@tanstack/solid-query";
import {
	type Accessor,
	createEffect,
	createMemo,
	createSignal,
	on,
	onCleanup,
	onMount,
} from "solid-js";
import { createStore, produce, reconcile, unwrap } from "solid-js/store";
import { z } from "zod";
import { keys } from "@/api/keys";
import { saveDraftSlots } from "@/api/tailoring";
import { blockText } from "@/lib/docLayout";
import { ADDED_PREFIX, isAddedSlot, slotsToSave } from "@/lib/draftSlots";
import { createSaveLoop, type SaveStatus } from "@/lib/saveLoop";
import { reviewFindings } from "@/lib/tailoring";
import type { Draft, DraftLayout, LayoutBlock } from "@/types/tailoring";

export const PROFILE_SLOT = "profile";

const SAVE_DELAY_MS = 1500;
const PAGE_COUNT = /is (\d+) pages?/;

const oneLine = (text: string) => text.replace(/\s*\n\s*/g, " ");

function resolvedKey(id: string) {
	return `draft.resolved.${id}`;
}

const resolvedSchema = z.record(z.string(), z.boolean());

function readResolved(id: string): Record<string, boolean> {
	try {
		return resolvedSchema.parse(
			JSON.parse(localStorage.getItem(resolvedKey(id)) ?? "{}"),
		);
	} catch {
		return {};
	}
}

export function createDraftEditor(
	draft: Accessor<Draft>,
	layout: Accessor<DraftLayout | null>,
) {
	const queryClient = useQueryClient();
	const id = draft().id;
	const [blocks, setBlocks] = createSignal<LayoutBlock[]>(
		layout()?.blocks ?? [],
	);
	const isSlotBlock = (b: LayoutBlock) => b.slotId && !b.section;
	const slotIds = () =>
		blocks()
			.filter(isSlotBlock)
			.map((b) => b.slotId);
	const slotTexts = (list: LayoutBlock[]): Record<string, string> =>
		Object.fromEntries(
			list.filter(isSlotBlock).map((b) => [b.slotId, blockText(b)]),
		);
	const original = slotTexts(layout()?.blocks ?? []);

	const [texts, setTexts] = createStore<Record<string, string>>({
		...original,
	});
	const [saved, setSaved] = createStore<Record<string, string>>({
		...original,
	});

	createEffect(
		on(
			layout,
			(next) => {
				if (!next) return;
				for (const [slotId, t] of Object.entries(slotTexts(next.blocks))) {
					if (slotId in texts) continue;
					original[slotId] = t;
					setTexts(slotId, t);
					setSaved(slotId, t);
				}
				const merged = [...next.blocks];
				const local = blocks();
				local.forEach((b, i) => {
					if (
						!isAddedSlot(b.slotId) ||
						merged.some((m) => m.slotId === b.slotId)
					)
						return;
					const anchor = merged.findIndex(
						(m) => m.slotId === local[i - 1]?.slotId,
					);
					merged.splice(anchor + 1, 0, b);
				});
				setBlocks(merged);
			},
			{ defer: true },
		),
	);
	const [removed, setRemoved] = createSignal<string[]>([]);
	const [status, setStatus] = createSignal<SaveStatus>("saved");
	const [googleSaved, setGoogleSaved] = createSignal(false);
	const [resolved, setResolvedMap] = createSignal(readResolved(id));

	const labels = createMemo(() => {
		const out: Record<string, string> = { [PROFILE_SLOT]: "Profile" };
		for (const p of draft().provenance?.positions ?? [])
			p.bullets.forEach((b, i) => {
				out[b.slotId] = `${p.title} bullet ${i + 1}`;
			});
		return out;
	});
	const label = (slotId: string) =>
		labels()[slotId] ?? (isAddedSlot(slotId) ? "New bullet" : "Line");

	const text = (slotId: string) => texts[slotId] ?? "";

	const send = async () => {
		const sent = { ...unwrap(texts) };
		const gone = removed();
		const slots = slotsToSave(slotIds(), sent, saved, gone);
		const result = await saveDraftSlots(id, slots);
		setSaved(
			reconcile(
				Object.fromEntries(
					slots
						.filter((s) => !s.remove)
						.map((s) => [s.slotId, sent[s.slotId] ?? ""]),
				),
			),
		);
		setRemoved((now) => now.filter((r) => !gone.includes(r)));
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

	const addBullet = (afterId: string): string | undefined => {
		const list = blocks();
		const at = list.findIndex((b) => b.slotId === afterId);
		const anchor = list[at];
		if (!anchor?.bullet || anchor.section || afterId === PROFILE_SLOT) return;
		if (text(afterId).trim() === "") return;
		const slotId = `${ADDED_PREFIX}${crypto.randomUUID()}`;
		const lead = anchor.runs[0];
		const block: LayoutBlock = {
			...anchor,
			slotId,
			runs: lead ? [{ ...lead, text: " " }] : [],
		};
		setTexts(slotId, "");
		setBlocks([...list.slice(0, at + 1), block, ...list.slice(at + 1)]);
		return slotId;
	};

	const removeBullet = (slotId: string): string | undefined => {
		if (!isAddedSlot(slotId)) return;
		const list = blocks();
		const at = list.findIndex((b) => b.slotId === slotId);
		const before = list.slice(0, at).filter(isSlotBlock).at(-1);
		if (at < 0 || !before) return;
		setBlocks(list.filter((_, i) => i !== at));
		setTexts(
			produce((t) => {
				delete t[slotId];
			}),
		);
		if (slotId in saved) setRemoved((now) => [...now, slotId]);
		loop.schedule();
		return before.slotId;
	};

	const dirty = () =>
		removed().length > 0 || slotIds().some((s) => text(s) !== (saved[s] ?? ""));
	const pageCountFinding = () =>
		draft().findings.find((f) => f.check === "page_count");
	const googlePages = (): number | undefined => {
		const finding = pageCountFinding();
		if (dirty() || !finding) return undefined;
		const n = Number(PAGE_COUNT.exec(finding.message)?.[1]);
		return n > 1 ? n : 2;
	};

	const googleAgrees = () => googleSaved() && !dirty() && !pageCountFinding();

	const findingsFor = (slotId: string) =>
		slotId === PROFILE_SLOT
			? []
			: reviewFindings(draft().findings).filter((f) => f.slotId === slotId);
	const cardKeys = (hasCard: (slotId: string) => boolean) =>
		slotIds().filter((slotId) => hasCard(slotId) || findingsFor(slotId).length);

	const setResolved = (key: string, value: boolean) => {
		const next = { ...resolved(), [key]: value };
		setResolvedMap(next);
		try {
			localStorage.setItem(resolvedKey(id), JSON.stringify(next));
		} catch {}
	};

	return {
		slotIds,
		blocks,
		addBullet,
		removeBullet,
		label,
		text,
		original: (slotId: string) => original[slotId] ?? "",
		setText,
		undo: (slotId: string) => {
			if (!isAddedSlot(slotId)) setText(slotId, original[slotId] ?? "");
			else if (removeBullet(slotId) === undefined) setText(slotId, "");
		},
		edited: (slotId: string) => text(slotId) !== original[slotId],
		findingsFor,
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
