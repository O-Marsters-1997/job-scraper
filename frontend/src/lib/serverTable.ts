export function pageCount(total: number, pageSize: number): number {
	return Math.max(1, Math.ceil(total / pageSize));
}

export function clampPage(
	page: number,
	total: number,
	pageSize: number,
): number {
	return Math.min(page, pageCount(total, pageSize));
}

export function filterSignature(state: object): string {
	return JSON.stringify(state, (key, value) =>
		key === "page" ? undefined : value,
	);
}

export function nextSelection<Item>(
	previous: Record<string, Item>,
	next: Record<string, boolean>,
	pageItems: Item[],
	rowId: (item: Item) => string,
): Record<string, Item> {
	const onPage = new Map(pageItems.map((item) => [rowId(item), item]));
	const kept: Record<string, Item> = {};
	for (const [id, isOn] of Object.entries(next)) {
		const item = isOn ? (previous[id] ?? onPage.get(id)) : undefined;
		if (item !== undefined) kept[id] = item;
	}
	return kept;
}
