import type { Achievement, AchievementStats } from "@/types/experience";

export type AchievementSort = "bank" | "usage" | "rate";

const RATE_MIN_APPLIED = 5;

export function statLine(stats: AchievementStats | null): string {
	if (!stats || stats.inCvs === 0) return "";
	const parts = [`in ${stats.inCvs} CV${stats.inCvs === 1 ? "" : "s"}`];
	if (stats.leftOut > 0) parts.push(`left out ${stats.leftOut}× while higher`);
	if (stats.applied > 0) {
		const ratio = `${stats.replied}/${stats.applied} replied`;
		parts.push(
			stats.applied >= RATE_MIN_APPLIED
				? `${ratio} (${Math.round((stats.replied / stats.applied) * 100)}%)`
				: ratio,
		);
	}
	return parts.join(" · ");
}

function replyRate(stats: AchievementStats | null): number | null {
	if (!stats || stats.applied < RATE_MIN_APPLIED) return null;
	return stats.replied / stats.applied;
}

export function sortAchievements(
	items: Achievement[],
	sort: AchievementSort,
): Achievement[] {
	if (sort === "bank") return items;
	const usage = (a: Achievement) => a.stats?.inCvs ?? 0;
	return [...items].sort((a, b) => {
		if (sort === "rate") {
			const ra = replyRate(a.stats);
			const rb = replyRate(b.stats);
			if (ra !== null && rb !== null && ra !== rb) return rb - ra;
			if ((ra === null) !== (rb === null)) return ra === null ? 1 : -1;
		}
		return usage(b) - usage(a);
	});
}
