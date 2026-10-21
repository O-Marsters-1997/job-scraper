import assert from "node:assert/strict";
import type { Job } from "@/types/job";
import { firstFoundBy } from "./insights";

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
