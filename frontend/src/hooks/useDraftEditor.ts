import { useQueryClient } from "@tanstack/solid-query";
import {
	type Accessor,
	createMemo,
	createSignal,
	onCleanup,
	onMount,
} from "solid-js";
import { createStore, unwrap } from "solid-js/store";
import { z } from "zod";
import { keys } from "@/api/keys";
import { saveDraftSlots } from "@/api/tailoring";
import { blockText } from "@/lib/docLayout";
import { createSaveLoop, type SaveStatus } from "@/lib/saveLoop";
import { reviewFindings } from "@/lib/tailoring";
import type { Draft, DraftLayout } from "@/types/tailoring";

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
	layout: DraftLayout | null,
) {
	const queryClient = useQueryClient();
	const id = draft().id;
	const blocks = layout?.blocks ?? [];
	const slotBlocks = blocks.filter((b) => b.slotId && !b.section);
	const slotIds = slotBlocks.map((b) => b.slotId);
	const original: Record<string, string> = Object.fromEntries(
		slotBlocks.map((b) => [b.slotId, blockText(b)]),
	);

	const [texts, setTexts] = createStore<Record<string, string>>({
		...original,
	});
	const [saved, setSaved] = createStore<Record<string, string>>({
		...original,
	});
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
	const label = (slotId: string) => labels()[slotId] ?? "Line";

	const text = (slotId: string) => texts[slotId] ?? "";

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

	const findingsFor = (slotId: string) =>
		slotId === PROFILE_SLOT
			? []
			: reviewFindings(draft().findings).filter((f) => f.slotId === slotId);
	const cardKeys = (hasCard: (slotId: string) => boolean) =>
		slotIds.filter((slotId) => hasCard(slotId) || findingsFor(slotId).length);

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
