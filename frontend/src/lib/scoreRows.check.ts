import assert from "node:assert/strict";
import type { ScoreRow } from "@/types/job";
import { rowLabel } from "./scoreRows";

const row = (label: string, effect: ScoreRow["effect"]) =>
	({ label, effect }) as ScoreRow;

assert.equal(
	rowLabel(row("Seniority: job ≈ 2.8 · you 2.3 ± 0.5", "meets")),
	"Seniority fits: Senior",
);
assert.equal(
	rowLabel(row("Seniority: job ≈ 4.0 · you 2.3 ± 0.5", "gated")),
	"Seniority: Lead / Staff role, you aim for Mid",
);
assert.equal(rowLabel(row("Go", "meets")), "Go");
