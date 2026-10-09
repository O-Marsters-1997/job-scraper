import type { Quota, UsageLevel } from "../types/quota";

const RANK: Record<UsageLevel, number> = { ok: 0, warn: 1, critical: 2 };

export function worstLevel(
	levels: readonly (UsageLevel | undefined)[],
): UsageLevel {
	let worst: UsageLevel = "ok";
	for (const level of levels) {
		if (level && RANK[level] > RANK[worst]) worst = level;
	}
	return worst;
}

export function quotaLevel(quota: Quota | undefined): UsageLevel {
	return quota?.status === "ok" ? quota.level : "ok";
}
