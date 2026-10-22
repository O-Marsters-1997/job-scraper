import { type Accessor, createMemo, createSignal, untrack } from "solid-js";
import type { Draft, LayoutBlock, Suggestion } from "@/types/tailoring";
import type { DraftEditorState } from "./useDraftEditor";

export function createRolePicks(opts: {
	draft: Accessor<Draft>;
	editor: DraftEditorState;
	bank: Accessor<Suggestion[]>;
}) {
	const roles = () => opts.draft().provenance?.positions ?? [];

	const [picks, setPicks] = createSignal<Record<string, string[]>>(
		untrack(() =>
			Object.fromEntries(
				roles().map((p) => [
					p.positionId,
					p.bullets.flatMap((b) => b.achievements.map((a) => a.id)),
				]),
			),
		),
	);

	const generated = untrack(() => {
		const out = new Map<string, string>();
		for (const p of opts.draft().content?.positions ?? [])
			for (const b of p.bullets) {
				const [only] = b.achievementIds;
				if (b.achievementIds.length === 1 && only && !out.has(only))
					out.set(only, b.text);
			}
		return out;
	});

	const headings = createMemo(() => {
		const list = opts.editor.blocks();
		const found = new Map<LayoutBlock, string>();
		for (const p of roles()) {
			const at = list.findIndex((b) => b.slotId === p.bullets[0]?.slotId);
			const head = list
				.slice(0, Math.max(at, 0))
				.reverse()
				.find((b) => !b.bullet && !b.slotId);
			if (at >= 0 && head) found.set(head, p.positionId);
		}
		return found;
	});

	const roleSlots = (positionId: string) => {
		const list = opts.editor.blocks();
		const head = [...headings()].find(([, id]) => id === positionId)?.[0];
		if (!head) return [];
		const slots: string[] = [];
		for (const b of list.slice(list.indexOf(head) + 1)) {
			if (!b.bullet || !b.slotId) break;
			slots.push(b.slotId);
		}
		return slots;
	};

	const bankFor = (positionId: string) =>
		opts
			.bank()
			.filter((s) => s.positionId === positionId)
			.sort((a, b) => b.score - a.score);

	const textOf = (id: string) =>
		generated.get(id) ??
		opts.bank().find((s) => s.achievementId === id)?.text ??
		roles()
			.flatMap((p) => p.bullets.flatMap((b) => b.achievements))
			.find((a) => a.id === id)?.text ??
		"";

	const setPicksFor = (positionId: string, ids: string[]) => {
		setPicks({ ...picks(), [positionId]: ids });
		if (ids.length === 0) return;
		opts.editor.replaceRole(
			positionId,
			roleSlots(positionId),
			ids.map((id) => ({
				text: textOf(id),
				achievementIds: [id],
			})),
		);
	};

	const picksFor = (positionId: string) => picks()[positionId] ?? [];

	return {
		headings,
		roleSlots,
		bankFor,
		picksFor,
		role: (positionId: string) =>
			roles().find((p) => p.positionId === positionId),
		toggle(positionId: string, id: string) {
			const now = picksFor(positionId);
			setPicksFor(
				positionId,
				now.includes(id) ? now.filter((x) => x !== id) : [...now, id],
			);
		},
		move(positionId: string, id: string, place: number) {
			const rest = picksFor(positionId).filter((x) => x !== id);
			rest.splice(Math.min(place - 1, rest.length), 0, id);
			setPicksFor(positionId, rest);
		},
		clear: (positionId: string) => setPicks({ ...picks(), [positionId]: [] }),
	};
}
