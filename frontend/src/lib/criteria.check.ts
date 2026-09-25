import { criterionScores, isLowConfidence } from "./criteria";

function ok(cond: boolean, msg: string): void {
	if (!cond) throw new Error(`FAIL: ${msg}`);
}
function eq(a: unknown, b: unknown, msg: string): void {
	if (JSON.stringify(a) !== JSON.stringify(b))
		throw new Error(
			`FAIL: ${msg} — expected ${JSON.stringify(b)}, got ${JSON.stringify(a)}`,
		);
}

eq(criterionScores(null), [], "null criteria yields no scores");
eq(criterionScores(undefined), [], "undefined criteria yields no scores");
eq(criterionScores({}), [], "empty criteria yields no scores");

eq(
	criterionScores({ go_backend: 0.5, remote: 0.49 }),
	[
		{ key: "go_backend", probability: 0.5, matched: true },
		{ key: "remote", probability: 0.49, matched: false },
	],
	"matches at the 0.5 boundary, misses just below it",
);

ok(!isLowConfidence(null), "null confidence is not flagged");
ok(!isLowConfidence(undefined), "undefined confidence is not flagged");
ok(!isLowConfidence(0.4), "confidence at the threshold is not flagged");
ok(isLowConfidence(0.39), "confidence just below the threshold is flagged");
ok(isLowConfidence(0), "zero confidence is flagged");
