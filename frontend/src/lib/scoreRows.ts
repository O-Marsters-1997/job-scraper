import type { ScoreRow } from "@/types/job";

const SENIORITY_LEVELS = [
	"Junior",
	"Mid",
	"Senior",
	"Lead / Staff",
	"Principal / Head",
];

const SENIORITY_LABEL = /^Seniority: job ≈ ([\d.]+) · you ([\d.]+)/;

const levelName = (n: string) =>
	SENIORITY_LEVELS[
		Math.min(Math.max(Math.round(Number(n)), 1), SENIORITY_LEVELS.length) - 1
	];

export function rowLabel(row: ScoreRow): string {
	const m = SENIORITY_LABEL.exec(row.label);
	if (!m?.[1] || !m[2]) return row.label;
	return row.effect === "meets"
		? `Seniority fits: ${levelName(m[1])}`
		: `Seniority: ${levelName(m[1])} role, you aim for ${levelName(m[2])}`;
}
