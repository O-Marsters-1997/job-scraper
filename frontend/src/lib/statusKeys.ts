import type { ApplicationStatus } from "../types/applicationStatus";

export const BOARD_NAV_KEYS: readonly string[] = ["h", "j", "k", "l"];

export function assignStatusKeys(
	statuses: Pick<ApplicationStatus, "ID" | "Name">[],
): Record<string, string> {
	const taken = new Set(BOARD_NAV_KEYS);
	const keys: Record<string, string> = {};
	for (const s of statuses) {
		const letter = [...s.Name.toLowerCase()].find(
			(c) => /[a-z]/.test(c) && !taken.has(c),
		);
		if (!letter) continue;
		taken.add(letter);
		keys[s.ID] = letter;
	}
	return keys;
}
