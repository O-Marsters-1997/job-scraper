const key = (text: string) => text.trim().replace(/\s+/g, " ").toLowerCase();

export function matchCarried(
	ids: string[],
	isCarried: (id: string) => boolean,
	textOf: (id: string) => string,
	bank: { achievementId: string; text: string }[],
): string[] {
	const taken = new Set(ids.filter((id) => !isCarried(id)));
	return ids.map((id) => {
		if (!isCarried(id)) return id;
		const want = key(textOf(id));
		const hit = bank.find(
			(s) => !taken.has(s.achievementId) && key(s.text) === want,
		);
		if (!hit) return id;
		taken.add(hit.achievementId);
		return hit.achievementId;
	});
}
