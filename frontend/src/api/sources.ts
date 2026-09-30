import { z } from "zod";
import { ResolveError } from "../lib/resolveError";
import {
	type ResolvedURL,
	resolvedUrlSchema,
	sourceInfoSchema,
} from "../types/source";
import { ApiError, apiFetch } from "./client";
import { mocked } from "./config";

const errorBodySchema = z.object({ error: z.string() });

export async function fetchSources() {
	return mocked(
		(db) => db.getSources(),
		() => apiFetch("/sources", sourceInfoSchema.array()),
	);
}

export async function resolveUrl(url: string): Promise<ResolvedURL> {
	return mocked(
		(db) => db.resolveUrl(url),
		() =>
			apiFetch(
				`/sources/resolve?${new URLSearchParams({ url })}`,
				resolvedUrlSchema,
			).catch((err) => {
				if (err instanceof ApiError && err.status === 422) {
					const body = errorBodySchema.safeParse(err.body);
					throw new ResolveError(
						body.success ? body.data.error : "Unrecognised URL",
					);
				}
				throw err;
			}),
	);
}
