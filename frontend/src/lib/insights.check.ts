import assert from "node:assert/strict";
import type { Job } from "@/types/job";
import { buildInsights, firstFoundBy } from "./insights";

const job = (patch: Partial<Job>) =>
	({ Source: "greenhouse", ...patch }) as Job;
const listing = (source: string, first_seen_at: string) => ({
	source,
	url: `https://example.com/${source}`,
	first_seen_at,
});

assert.equal(firstFoundBy(job({ DiscoveredVia: "wellfound" })), "wellfound");
assert.equal(
	firstFoundBy(
		job({
			DiscoveredVia: "wellfound",
			Listings: [listing("linkedin", "2026-10-01T00:00:00Z")],
		}),
	),
	"wellfound",
);
assert.equal(
	firstFoundBy(
		job({
			Listings: [
				listing("greenhouse", "2026-10-03T00:00:00Z"),
				listing("linkedin", "2026-10-01T00:00:00Z"),
			],
		}),
	),
	"linkedin",
);
assert.equal(
	firstFoundBy(job({ DiscoveredVia: "", Listings: [] })),
	"greenhouse",
);
assert.equal(firstFoundBy(job({ Listings: null })), "greenhouse");

const scored = (patch: Partial<Job>) =>
	job({ ScrapedAt: "2026-10-01T09:00:00Z", ...patch });
const insights = buildInsights([
	scored({ Source: "lever", SuitabilityScore: 85, Band: "great" }),
	scored({ Source: "lever", SuitabilityScore: 30, Band: "poor" }),
	scored({ Source: "ashby", SuitabilityScore: null }),
]);
assert.equal(insights.total, 3);
assert.equal(insights.scored, 2);
assert.deepEqual(insights.scoreBands, [0, 1, 0, 0, 1]);
assert.deepEqual(insights.sources, [
	{ source: "lever", count: 2, scored: 2, avgScore: 58 },
	{ source: "ashby", count: 1, scored: 0, avgScore: 0 },
]);
assert.deepEqual(insights.foundBy, [{ name: "lever", count: 1 }]);
