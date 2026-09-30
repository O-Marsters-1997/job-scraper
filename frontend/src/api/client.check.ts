import assert from "node:assert/strict";
import { z } from "zod";
import { ApiError, apiFetch } from "./client";
import { API_BASE } from "./config";

const originalFetch = globalThis.fetch;
let requested = "";
let credentials: RequestCredentials | undefined;
let response = () => Response.json({ id: 123 });
globalThis.fetch = async (input, init) => {
	requested = String(input);
	credentials = init?.credentials;
	return response();
};
try {
	await assert.rejects(apiFetch("/jobs", z.object({ id: z.string() })));
	assert.equal(requested, `${API_BASE}/jobs`);
	assert.equal(credentials, "include");

	response = () => Response.json({ error: "taken" }, { status: 409 });
	await assert.rejects(
		apiFetch("/jobs", z.unknown()),
		(err) =>
			err instanceof ApiError &&
			err.status === 409 &&
			(err.body as { error: string }).error === "taken",
	);
} finally {
	globalThis.fetch = originalFetch;
}
