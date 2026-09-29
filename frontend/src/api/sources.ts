import { z } from "zod";
import { ResolveError } from "../lib/resolveError";
import {
	type ResolvedURL,
	resolvedUrlSchema,
	sourceInfoSchema,
} from "../types/source";
import { apiFetch } from "./client";
import { API_BASE, mocked } from "./config";

export async function fetchSources() {
	return mocked(
		(db) => db.getSources(),
		() => apiFetch("/sources", undefined, sourceInfoSchema.array()),
	);
}

export async function resolveUrl(url: string): Promise<ResolvedURL> {
	return mocked(
		(db) => db.resolveUrl(url),
		async () => {
			const response = await fetch(
				`${API_BASE}/sources/resolve?${new URLSearchParams({ url })}`,
				{ credentials: "include" },
			);
			if (response.status === 422) {
				const body: unknown = await response.json().catch(() => null);
				const message = z.object({ error: z.string() }).safeParse(body);
				throw new ResolveError(
					message.success ? message.data.error : "Unrecognised URL",
				);
			}
			if (!response.ok) throw new Error(`resolve: ${response.status}`);
			return resolvedUrlSchema.parse(await response.json());
		},
	);
}
