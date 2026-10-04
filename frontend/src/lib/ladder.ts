export const TOLERANCE_FLOOR = 0.5;

export type LadderTier = { level: number; weight: number };

export type LadderBand = { point: number; tolerance: number };

export function ladderBand(tiers: LadderTier[]): LadderBand | null {
	const total = tiers.reduce((sum, t) => sum + t.weight, 0);
	if (total <= 0) return null;
	const point = tiers.reduce((sum, t) => sum + t.weight * t.level, 0) / total;
	const variance =
		tiers.reduce((sum, t) => sum + t.weight * (t.level - point) ** 2, 0) /
		total;
	return { point, tolerance: Math.max(TOLERANCE_FLOOR, Math.sqrt(variance)) };
}
