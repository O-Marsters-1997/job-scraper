import { useQueryClient } from "@tanstack/solid-query";
import {
	type Accessor,
	batch,
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
import type {
	Draft,
	DraftLayout,
	DraftPositionEdit,
	LayoutBlock,
} from "@/types/tailoring";

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

type RoleEdit = { slots: string[]; ids: string[][] };

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
	const [original, setOriginal] = createStore<Record<string, string>>(
		slotTexts(layout()?.blocks ?? []),
	);

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
					setOriginal(slotId, t);
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

	const [roleEdits, setRoleEdits] = createSignal<Record<string, RoleEdit>>({});

	const rekey = (
		result: Draft,
		sent: { positionId: string; slots: string[]; texts: string[] }[],
	) => {
		const renamed = new Map<string, string>();
		const sentTexts: Record<string, string> = {};
		for (const { positionId, slots, texts: sentAt } of sent) {
			const bullets =
				result.provenance?.positions.find((p) => p.positionId === positionId)
					?.bullets ?? [];
			slots.forEach((slot, i) => {
				const next = bullets[i]?.slotId ?? slot;
				renamed.set(slot, next);
				sentTexts[next] = sentAt[i] ?? "";
			});
		}
		const to = (slotId: string) => renamed.get(slotId) ?? slotId;
		const move = (record: Record<string, string>) =>
			Object.fromEntries(Object.entries(record).map(([k, v]) => [to(k), v]));
		setRoleEdits((now) =>
			Object.fromEntries(
				Object.entries(now).map(([k, v]) => [
					k,
					{ ...v, slots: v.slots.map(to) },
				]),
			),
		);
		setBlocks(
			blocks().map((b) =>
				to(b.slotId) === b.slotId ? b : { ...b, slotId: to(b.slotId) },
			),
		);
		setTexts(reconcile(move(unwrap(texts))));
		setOriginal(reconcile(move(unwrap(original))));
		const live = new Set([...slotIds(), ...removed()]);
		setSaved(
			reconcile(
				Object.fromEntries(
					Object.entries({ ...move(unwrap(saved)), ...sentTexts }).filter(
						([k]) => live.has(k),
					),
				),
			),
		);
	};

	const sentRoles = (edits: Record<string, RoleEdit>) =>
		Object.entries(edits).flatMap(([positionId, edit]) => {
			const kept = edit.slots.flatMap((slot, i) => {
				const body = text(slot).trim();
				return body ? [{ slot, body, ids: edit.ids[i] ?? [] }] : [];
			});
			if (kept.length === 0 && edit.ids.length === 0)
				return [{ positionId, slots: [], texts: [], bullets: [] }];
			return kept.length
				? [
						{
							positionId,
							slots: kept.map((k) => k.slot),
							texts: kept.map((k) => k.body),
							bullets: kept.map((k) => ({
								text: k.body,
								achievementIds: k.ids,
							})),
						},
					]
				: [];
		});

	type SentRole = ReturnType<typeof sentRoles>[number];

	const positionEdits = (sent: SentRole[]): DraftPositionEdit[] =>
		sent.map((s) => ({ positionId: s.positionId, bullets: s.bullets }));

	const settleRoles = (
		edits: Record<string, RoleEdit>,
		sent: SentRole[],
		result: Draft,
	) => {
		setRoleEdits((now) =>
			Object.fromEntries(
				Object.entries(now).filter(([k, v]) => edits[k] !== v),
			),
		);
		rekey(result, sent);
		for (const { positionId } of sent.filter((s) => s.bullets.length === 0)) {
			const [placeholder] = edits[positionId]?.slots ?? [];
			if (placeholder) setSaved(placeholder, "");
		}
		setGoogleSaved(true);
		queryClient.setQueryData(keys.tailoring.draft(id), result);
		void queryClient.invalidateQueries({
			queryKey: keys.tailoring.draftInsights(id),
		});
		void queryClient.invalidateQueries({
			queryKey: keys.tailoring.draftRoles(id),
		});
	};

	const sendRoles = async () => {
		const edits = roleEdits();
		const sent = sentRoles(edits);
		if (sent.length === 0) return false;
		const result = await saveDraftSlots(id, [], undefined, positionEdits(sent));
		batch(() => settleRoles(edits, sent, result));
		return true;
	};

	const [pendingRoles, setPendingRoles] = createSignal<string[]>();

	const sendOrder = async () => {
		const order = pendingRoles();
		if (!order) return;
		const known = new Set(
			draft().provenance?.positions.map((p) => p.positionId),
		);
		const edits = Object.fromEntries(
			Object.entries(roleEdits()).filter(
				([k]) => order.includes(k) && !known.has(k),
			),
		);
		const sent = sentRoles(edits);
		const result = await saveDraftSlots(
			id,
			[],
			undefined,
			sent.length > 0 ? positionEdits(sent) : undefined,
			order,
		);
		batch(() => {
			setPendingRoles((now) => (now === order ? undefined : now));
			settleRoles(edits, sent, result);
		});
	};

	const send = async () => {
		const reordered = pendingRoles() !== undefined;
		await sendOrder();
		if (((await sendRoles()) || reordered) && !dirty()) return;
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
		void queryClient.invalidateQueries({
			queryKey: keys.tailoring.draftInsights(id),
		});
		void queryClient.invalidateQueries({
			queryKey: keys.tailoring.draftRoles(id),
		});
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

	const replaceRole = (
		positionId: string,
		slots: string[],
		bullets: DraftPositionEdit["bullets"],
	) => {
		const list = blocks();
		const lastAt = list.findIndex((b) => b.slotId === slots.at(-1));
		const template = list.find((b) => b.slotId === slots[0]);
		if (!template || lastAt < 0) return;
		const keep = Math.max(bullets.length, 1);
		const dropped = new Set(slots.slice(keep));
		const fresh = bullets
			.slice(slots.length)
			.map(() => `${ADDED_PREFIX}${crypto.randomUUID()}`);
		const final = [...slots.slice(0, keep), ...fresh];
		const lead = template.runs[0];
		setBlocks([
			...list.slice(0, lastAt + 1).filter((b) => !dropped.has(b.slotId)),
			...fresh.map((slotId) => ({
				...template,
				slotId,
				runs: lead ? [{ ...lead, text: " " }] : [],
			})),
			...list.slice(lastAt + 1),
		]);
		const body = (i: number) => oneLine(bullets[i]?.text ?? "");
		setOriginal(
			produce((o) => {
				for (const s of dropped) delete o[s];
				final.forEach((s, i) => {
					o[s] = body(i);
				});
			}),
		);
		setTexts(
			produce((t) => {
				for (const s of dropped) delete t[s];
				final.forEach((s, i) => {
					t[s] = body(i);
				});
			}),
		);
		setRoleEdits((now) => ({
			...now,
			[positionId]: { slots: final, ids: bullets.map((b) => b.achievementIds) },
		}));
		loop.schedule();
	};

	const rearrange = (next: LayoutBlock[], order: string[]) => {
		const kept = new Set(next.map((b) => b.slotId));
		const gone = slotIds().filter((s) => !kept.has(s));
		setBlocks(next);
		for (const store of [setTexts, setSaved])
			store(
				produce((t) => {
					for (const s of gone) delete t[s];
				}),
			);
		setRoleEdits((now) =>
			Object.fromEntries(
				Object.entries(now).filter(([positionId]) =>
					order.includes(positionId),
				),
			),
		);
		setPendingRoles(order);
		loop.schedule();
	};

	const dirty = () =>
		pendingRoles() !== undefined ||
		removed().length > 0 ||
		slotIds().some((s) => text(s) !== (saved[s] ?? ""));
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
		replaceRole,
		rearrange,
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
