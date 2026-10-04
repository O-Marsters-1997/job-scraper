import assert from "node:assert/strict";
import { ladderBand } from "./ladder";

const close = (got: number, want: number, what: string) =>
	assert.ok(Math.abs(got - want) < 0.01, `${what} = ${got}, want ${want}`);

const tight = ladderBand([
	{ level: 2, weight: 70 },
	{ level: 3, weight: 30 },
]);
assert.ok(tight);
close(tight.point, 2.3, "Mid 70 / Senior 30 point");
close(tight.tolerance, 0.5, "Mid 70 / Senior 30 tolerance, floored");

const wide = ladderBand([
	{ level: 1, weight: 15 },
	{ level: 2, weight: 55 },
	{ level: 3, weight: 15 },
	{ level: 4, weight: 15 },
]);
assert.ok(wide);
close(wide.point, 2.3, "wide point");
close(wide.tolerance, 0.9, "wide tolerance");

assert.equal(ladderBand([{ level: 3, weight: 0 }]), null, "no weight, no band");
assert.equal(ladderBand([]), null, "no tiers, no band");
