export type CardSlot = { key: string; desired: number; height: number };

export function cardTops(
	cards: CardSlot[],
	pinned: number,
	gap: number,
): Record<string, number> {
	const tops: Record<string, number> = {};
	if (cards.length === 0) return tops;
	const at = Math.min(Math.max(pinned, 0), cards.length - 1);
	let cursor = Number.NEGATIVE_INFINITY;
	for (const c of cards.slice(at)) {
		const top = Math.max(c.desired, cursor);
		tops[c.key] = top;
		cursor = top + c.height + gap;
	}
	cursor = tops[cards[at]?.key ?? ""] ?? 0;
	for (const c of cards.slice(0, at).reverse()) {
		const top = Math.min(c.desired, cursor - c.height - gap);
		tops[c.key] = top;
		cursor = top;
	}
	return tops;
}
