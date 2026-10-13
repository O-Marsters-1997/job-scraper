import assert from "node:assert/strict";
import { postedLabel } from "./postedLabel";

const savedTZ = process.env.TZ;
try {
	process.env.TZ = "Europe/London";

	const DAY = 24 * 3600 * 1000;
	const ago = (ms: number) => new Date(Date.now() - ms).toISOString();
	const base = {
		PostedAt: null,
		PostedAfter: null,
		RepostedAt: null,
		FirstDiscoveredAt: "2025-10-12T15:00:00.000Z",
		ScrapedAt: "2025-10-12T15:00:00.000Z",
	};

	const exact = postedLabel({ ...base, PostedAt: ago(3 * DAY) });
	assert.equal(exact.list, "3 days ago");
	assert.match(exact.detail, /^Posted \d{1,2} \w{3}$/);
	assert.equal(exact.detailRepost, undefined);

	const dateOnly = postedLabel({ ...base, PostedAt: "2025-10-12T00:00:00Z" });
	assert.equal(dateOnly.list, "12 Oct");
	assert.equal(dateOnly.detail, "Posted 12 Oct");

	process.env.TZ = "America/New_York";
	assert.equal(
		postedLabel({ ...base, PostedAt: "2025-10-12T00:00:00Z" }).detail,
		"Posted 12 Oct",
	);
	process.env.TZ = "Europe/London";

	const inverted = postedLabel({
		...base,
		PostedAfter: "2025-10-12T20:00:00.000Z",
	});
	assert.equal(inverted.detail, "Posted on or before 12 Oct");

	const range = postedLabel({
		...base,
		PostedAfter: "2025-10-12T09:00:00.000Z",
	});
	assert.match(range.list, /^~/);
	assert.equal(range.detail, "Posted between 12 Oct 10:00 and 16:00");
	assert.equal(range.listTitle, range.detail);

	const multiDay = postedLabel({
		...base,
		PostedAfter: "2025-10-11T09:00:00.000Z",
	});
	assert.equal(multiDay.detail, "Posted between 11 Oct 10:00 and 12 Oct 16:00");

	const neither = postedLabel(base);
	assert.equal(neither.list, "Before 12 Oct");
	assert.equal(neither.detail, "Posted on or before 12 Oct");
	assert.equal(neither.listTitle, undefined);

	const reposted = postedLabel({
		...base,
		PostedAt: ago(10 * DAY),
		RepostedAt: ago(2 * DAY),
	});
	assert.ok(reposted.list.endsWith(" · Reposted 2 days ago"));
	assert.match(reposted.detailRepost ?? "", /^Reposted \d{1,2} \w{3}$/);
} finally {
	if (savedTZ === undefined) delete process.env.TZ;
	else process.env.TZ = savedTZ;
}

console.log("✓ postedLabel checks passed");
