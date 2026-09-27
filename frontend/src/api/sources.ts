import {
	type ResolvedBoard,
	resolvedBoardSchema,
	sourceInfoSchema,
} from "../types/source";
import { apiFetch } from "./client";
import { API_BASE, useMocks } from "./config";

export async function fetchSources() {
	if (useMocks()) {
		const { getSources } = await import("../mocks/db");
		return getSources();
	}
	return apiFetch("/sources", undefined, sourceInfoSchema.array());
}

export async function resolveBoard(url: string): Promise<ResolvedBoard | null> {
	if (useMocks()) {
		const { resolveBoard: mockResolve } = await import("../mocks/db");
		return mockResolve(url);
	}
	const response = await fetch(
		`${API_BASE}/sources/resolve?${new URLSearchParams({ url })}`,
		{ credentials: "include" },
	);
	if (response.status === 422 || response.status === 400) return null;
	if (!response.ok) throw new Error(`resolve: ${response.status}`);
	return resolvedBoardSchema.parse(await response.json());
}
