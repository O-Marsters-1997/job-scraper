import assert from "node:assert/strict";
import { setDemoData } from "../lib/demoData";
import { auth, fetchMe, logout } from "./auth";
import { mocked } from "./config";

const originalFetch = globalThis.fetch;
const fetched: string[] = [];
globalThis.fetch = (async (input: RequestInfo | URL) => {
	fetched.push(String(input));
	return Response.json({ id: "real", username: "real" });
}) as typeof fetch;

try {
	setDemoData(true);
	const fromMock = await mocked(
		(db) => db.getJobs().length,
		() => -1,
	);
	assert.ok(fromMock >= 0, "demo mode must run the mock branch");
	assert.deepEqual(fetched, [], "demo mode must not reach fetch");
	assert.equal((await fetchMe()).username, "demo");
	assert.deepEqual(fetched, [], "fetchMe must not reach fetch in demo mode");

	setDemoData(false);
	assert.equal(
		await mocked(
			() => "mock",
			() => "real",
		),
		"real",
	);

	await auth("/auth/login", "u", "p");
	await logout();
	assert.equal(fetched.length, 2, "real mode auth and logout must hit fetch");
} finally {
	globalThis.fetch = originalFetch;
	setDemoData(false);
}

console.log("✓ mocks checks passed");
