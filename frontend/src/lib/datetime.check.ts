import assert from "node:assert/strict";
import { dayKey } from "./datetime";

const savedTZ = process.env.TZ;
try {
	process.env.TZ = "Europe/London";

	// 15 Jul 2025 is BST (UTC+1). 00:30 local is 23:30 the previous day in UTC.
	const scrapedAt = "2025-07-14T23:30:00.000Z";
	const localMidnight = new Date(2025, 6, 15, 0, 0, 0);

	assert.ok(
		dayKey(scrapedAt) === dayKey(localMidnight),
		"a job scraped at 00:30 local BST must land in today's local bucket",
	);
	assert.ok(dayKey(scrapedAt) === "2025-07-15");
} finally {
	if (savedTZ === undefined) delete process.env.TZ;
	else process.env.TZ = savedTZ;
}

console.log("✓ datetime checks passed");
