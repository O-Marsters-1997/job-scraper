import {
	type ResolvedBoard,
	resolvedBoardSchema,
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

export async function resolveBoard(url: string): Promise<ResolvedBoard | null> {
	return mocked(
		(db) => db.resolveBoard(url),
		async () => {
			const response = await fetch(
				`${API_BASE}/sources/resolve?${new URLSearchParams({ url })}`,
				{ credentials: "include" },
			);
			if (response.status === 422 || response.status === 400) return null;
			if (!response.ok) throw new Error(`resolve: ${response.status}`);
			return resolvedBoardSchema.parse(await response.json());
		},
	);
}
