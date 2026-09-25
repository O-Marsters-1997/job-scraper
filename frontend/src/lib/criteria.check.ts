import { criterionScores, isLowConfidence } from "./criteria";

// ponytail: inline assert so tsgo doesn't need @types/node
function ok(cond: boolean, msg?: string): void {
	if (!cond) throw new Error(msg ?? "assertion failed");
}
function eq(a: unknown, b: unknown): void {
	if (JSON.stringify(a) !== JSON.stringify(b))
		throw new Error(
			`expected ${JSON.stringify(a)} to equal ${JSON.stringify(b)}`,
		);
}

eq(criterionScores(null), []);
eq(criterionScores(undefined), []);
eq(criterionScores({}), []);

eq(criterionScores({ go_backend: 0.5, remote: 0.49 }), [
	{ key: "go_backend", probability: 0.5, matched: true },
	{ key: "remote", probability: 0.49, matched: false },
]);

ok(!isLowConfidence(null), "null confidence is not flagged");
ok(!isLowConfidence(undefined), "undefined confidence is not flagged");
ok(!isLowConfidence(0.4), "confidence at the threshold is not flagged");
ok(isLowConfidence(0.39), "confidence just below the threshold is flagged");
ok(isLowConfidence(0), "zero confidence is flagged");

console.log("✓ criteria checks passed");
