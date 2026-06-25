import type { SourceInfo } from "../types/source";
import { apiFetch } from "./client";

export async function fetchSources(): Promise<SourceInfo[]> {
	return apiFetch<SourceInfo[]>("/sources");
}
