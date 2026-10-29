import { type Accessor, createMemo, createSignal, untrack } from "solid-js";
import { ADDED_PREFIX } from "@/lib/draftSlots";
import { cloneRoleHeading } from "@/lib/roleHeading";
import type { Position } from "@/types/experience";
import type { Draft, LayoutBlock, Suggestion } from "@/types/tailoring";
import type { DraftEditorState } from "./useDraftEditor";

type AddedRole = { positionId: string; head: LayoutBlock };

const CARRIED = "carried:";
const carriedId = (positionId: string, at: number) =>
	`${CARRIED}${positionId}:${at}`;

export function createRolePicks(opts: {
	draft: Accessor<Draft>;
	editor: DraftEditorState;
	bank: Accessor<Suggestion[]>;
	positions: Accessor<Position[]>;
}) {
	const roles = () => opts.draft().provenance?.positions ?? [];
	const [added, setAdded] = createSignal<AddedRole[]>([]);

	const carried = untrack(() => {
		const out = new Map<string, string>();
		for (const c of opts.draft().content?.positions ?? [])
			c.bullets.forEach((b, i) => {
				if (b.achievementIds.length === 0 && b.text.trim())
					out.set(carriedId(c.positionId, i), b.text);
			});
		return out;
	});

	const initial = untrack(() =>
		Object.fromEntries(
			roles().map((p) => [
				p.positionId,
				p.bullets.flatMap((b, i) =>
					b.achievements.length > 0
						? b.achievements.map((a) => a.id)
						: carried.has(carriedId(p.positionId, i))
							? [carriedId(p.positionId, i)]
							: [],
				),
			]),
		),
	);
	const [picks, setPicks] = createSignal<Record<string, string[]>>(initial);

	const generated = untrack(() => {
		const out = new Map(carried);
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
		for (const a of added())
			if (list.includes(a.head)) found.set(a.head, a.positionId);
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

	const onPage = createMemo(() => {
		const list = opts.editor.blocks();
		const found = [...headings()]
			.map(([head, id]) => ({ id, start: list.indexOf(head) }))
			.sort((a, b) => a.start - b.start);
		return found.map((r, i) => {
			const lastSlot = list.findIndex(
				(b) => b.slotId === roleSlots(r.id).at(-1),
			);
			return { ...r, end: found[i + 1]?.start ?? lastSlot + 1 };
		});
	});

	const arrange = (order: string[]) => {
		const list = opts.editor.blocks();
		const groups = onPage();
		const first = groups[0];
		const last = groups.at(-1);
		if (!first || !last) return;
		opts.editor.rearrange(
			[
				...list.slice(0, first.start),
				...order.flatMap((id) => {
					const g = groups.find((r) => r.id === id);
					return g ? list.slice(g.start, g.end) : [];
				}),
				...list.slice(last.end),
			],
			order,
		);
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

	const setPicksFor = (positionId: string, ids: string[], fresh = false) => {
		const slots = roleSlots(positionId);
		const now = picksFor(positionId);
		setPicks({ ...picks(), [positionId]: ids });
		opts.editor.replaceRole(
			positionId,
			slots,
			ids.map((id) => {
				const slot = fresh ? undefined : slots[now.indexOf(id)];
				return {
					text: slot ? opts.editor.text(slot) : textOf(id),
					achievementIds: id.startsWith(CARRIED) ? [] : [id],
				};
			}),
		);
	};

	const picksFor = (positionId: string) => picks()[positionId] ?? [];
	const order = () => onPage().map((r) => r.id);

	const addRole = (position: Position) => {
		const list = opts.editor.blocks();
		const groups = onPage();
		const template = groups[0];
		const from = roles().find((p) => p.positionId === template?.id);
		const templateHead = [...headings()].find(
			([, id]) => id === template?.id,
		)?.[0];
		const templateBullet = list.find(
			(b) => b.slotId === roleSlots(template?.id ?? "")[0],
		);
		const best = bankFor(position.id)[0];
		const first = best
			? { id: best.achievementId, text: best.text }
			: position.achievements[0];
		if (!template || !from || !templateHead || !templateBullet || !first)
			return;
		const startOf = (id: string) =>
			opts.positions().find((p) => p.id === id)?.startDate ?? "";
		const newer = groups
			.filter((g) => startOf(g.id) > (position.startDate ?? ""))
			.at(-1);
		const at = newer ? newer.end : template.start;
		const head = cloneRoleHeading(templateHead, from, position);
		const slotId = `${ADDED_PREFIX}${crypto.randomUUID()}`;
		const lead = templateBullet.runs[0];
		const ids = groups.map((g) => g.id);
		ids.splice(newer ? ids.indexOf(newer.id) + 1 : 0, 0, position.id);
		setAdded([...added(), { positionId: position.id, head }]);
		setPicks({ ...picks(), [position.id]: [first.id] });
		opts.editor.rearrange(
			[
				...list.slice(0, at),
				head,
				{
					...templateBullet,
					slotId,
					runs: lead ? [{ ...lead, text: " " }] : [],
				},
				...list.slice(at),
			],
			ids,
		);
		opts.editor.replaceRole(
			position.id,
			[slotId],
			[{ text: first.text, achievementIds: [first.id] }],
		);
	};

	return {
		headings,
		roleSlots,
		bankFor,
		picksFor,
		role: (positionId: string) => {
			const saved = roles().find((p) => p.positionId === positionId);
			if (saved) return saved;
			const position = opts.positions().find((p) => p.id === positionId);
			return position && added().some((a) => a.positionId === positionId)
				? { title: position.title, employer: position.employer }
				: undefined;
		},
		addRole,
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
		shift(slotId: string, delta: -1 | 1) {
			const positionId = [...headings().values()].find((id) =>
				roleSlots(id).includes(slotId),
			);
			if (!positionId) return undefined;
			const ids = picksFor(positionId);
			const slots = roleSlots(positionId);
			const from = slots.indexOf(slotId);
			const to = from + delta;
			const id = ids[from];
			if (slots.length !== ids.length || id === undefined) return undefined;
			if (to < 0 || to >= ids.length) return undefined;
			const next = ids.filter((x) => x !== id);
			next.splice(to, 0, id);
			setPicksFor(positionId, next);
			return slots[to];
		},
		order,
		toggleRole(positionId: string) {
			const ids = order();
			if (!ids.includes(positionId)) {
				const position = opts.positions().find((p) => p.id === positionId);
				if (position) addRole(position);
			} else if (ids.length > 1) {
				arrange(ids.filter((id) => id !== positionId));
			}
		},
		moveRole(positionId: string, place: number) {
			const rest = order().filter((id) => id !== positionId);
			if (rest.length === order().length) return;
			rest.splice(Math.min(place - 1, rest.length), 0, positionId);
			arrange(rest);
		},
		carriedFor: (positionId: string) =>
			(initial[positionId] ?? [])
				.filter((id) => id.startsWith(CARRIED))
				.map((id) => ({ id, text: generated.get(id) ?? "" })),
		isPlaceholder: (slotId: string) =>
			[...headings().values()].some(
				(id) => picksFor(id).length === 0 && roleSlots(id)[0] === slotId,
			),
		empty: (positionId: string) => setPicksFor(positionId, []),
		reset: (positionId: string) =>
			setPicksFor(positionId, initial[positionId] ?? [], true),
	};
}
