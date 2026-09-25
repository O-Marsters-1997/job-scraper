import { z } from "zod";
import { apiFetch } from "./client";
import { API_BASE } from "./config";

const originalFetch = globalThis.fetch;
let requested = "";
let credentials: RequestCredentials | undefined;
globalThis.fetch = async (input, init) => {
	requested = String(input);
	credentials = init?.credentials;
	return Response.json({ id: 123 });
};
try {
	let rejected = false;
	try {
		await apiFetch("/jobs", undefined, z.object({ id: z.string() }));
	} catch {
		rejected = true;
	}
	if (
		!rejected ||
		requested !== `${API_BASE}/jobs` ||
		credentials !== "include"
	)
		throw new Error("apiFetch must validate responses and send credentials");
} finally {
	globalThis.fetch = originalFetch;
}
