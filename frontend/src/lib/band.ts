import type { GradeValue } from "@/types/grade";
import type { Band } from "@/types/job";
import {
	GOOD_COLOUR,
	MATCHED_COLOUR,
	MISSING_COLOUR,
	WARNING_COLOUR,
} from "./scoreColour";

export const BAND_LABEL: Record<Band, string> = {
	great: "Great",
	good: "Good",
	fair: "Fair",
	poor: "Poor",
};

export const BAND_COLOUR: Record<Band, string> = {
	great: MATCHED_COLOUR,
	good: GOOD_COLOUR,
	fair: WARNING_COLOUR,
	poor: MISSING_COLOUR,
};

const RANK: Record<Band, number> = { poor: 0, fair: 1, good: 2, great: 3 };

const AGREES: Record<GradeValue, [Band, Band]> = {
	great: ["good", "great"],
	ok: ["fair", "good"],
	no: ["poor", "fair"],
};

export type Direction = "too high" | "too low" | "about right";

export function gradeDirection(grade: GradeValue, band: Band): Direction {
	const [low, high] = AGREES[grade];
	if (RANK[band] > RANK[high]) return "too high";
	if (RANK[band] < RANK[low]) return "too low";
	return "about right";
}
