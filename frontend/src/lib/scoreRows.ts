import type { ScoreRow } from "@/types/job";

export function unknownCount(rows: ScoreRow[] | null | undefined): number {
	if (!rows) return 0;
	return rows.filter((r) => r.effect === "unknown").length;
}
