import type { SourceInfo } from "../types/source";
import { API_BASE } from "./config";

export async function fetchSources(): Promise<SourceInfo[]> {
	const res = await fetch(`${API_BASE}/sources`, {
		credentials: "include",
	});
	if (!res.ok) throw new Error(`Failed to fetch sources: ${res.status}`);
	return res.json();
}
