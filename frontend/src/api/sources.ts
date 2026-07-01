import type { SourceInfo } from "../types/source";
import { apiFetch } from "./client";
import { API_BASE } from "./config";

export async function fetchSources(): Promise<SourceInfo[]> {
	return apiFetch<SourceInfo[]>("/sources");
}

export interface ResolvedBoard {
	source: string;
	value: string;
}

// resolveBoard turns a pasted ATS board URL into a {source, value} pair, or null
// if the URL isn't a recognised ATS board (422/400).
export async function resolveBoard(url: string): Promise<ResolvedBoard | null> {
	const res = await fetch(
		`${API_BASE}/sources/resolve?url=${encodeURIComponent(url)}`,
		{ credentials: "include" },
	);
	if (res.status === 422 || res.status === 400) return null;
	if (!res.ok) throw new Error(`resolve: ${res.status}`);
	return res.json();
}
