import assert from "node:assert/strict";
import type { Job } from "@/types/job";
import type { SourceInfo } from "@/types/source";
import { otherListings, sourceLabel } from "./listings";

const primary = {
	source: "greenhouse",
	url: "https://boards.greenhouse.io/acme/jobs/1",
	first_seen_at: "2026-10-01T00:00:00Z",
};
const secondary = {
	source: "linkedin",
	url: "https://www.linkedin.com/jobs/view/1",
	first_seen_at: "2026-10-02T00:00:00Z",
};
const job = (listings: Job["Listings"]) =>
	({ URL: primary.url, Listings: listings }) as Job;

assert.deepEqual(otherListings(job([primary])), []);
assert.deepEqual(otherListings(job(undefined)), []);
assert.deepEqual(otherListings(job(null)), []);
assert.deepEqual(otherListings(job([primary, secondary])), [secondary]);

const sources: SourceInfo[] = [
	{
		name: "linkedin",
		label: "LinkedIn",
		kind: "filter",
		role: "discovery",
		url_prefix: "",
		filters: [],
	},
];
assert.equal(sourceLabel("linkedin", sources), "LinkedIn");
assert.equal(sourceLabel("work-in-startups", undefined), "Work In Startups");
