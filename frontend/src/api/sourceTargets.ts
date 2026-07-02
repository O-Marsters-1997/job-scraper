import type { SourceTarget } from "../types/sourceTarget";
import { apiFetch, apiFetchVoid } from "./client";
import { API_BASE, mockDelay, useMocks } from "./config";

export interface CreateSourceTargetPayload {
	source: string;
	value: string;
	enabled?: boolean;
	filters?: Record<string, string>;
	scrape_now?: boolean;
}

export class ConflictError extends Error {
	constructor() {
		super("source target already exists");
	}
}

export async function fetchSourceTargets(): Promise<SourceTarget[]> {
	return apiFetch<SourceTarget[]>("/source-targets");
}

export async function createSourceTarget(
	payload: CreateSourceTargetPayload,
): Promise<SourceTarget> {
	const res = await fetch(`${API_BASE}/source-targets`, {
		method: "POST",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	});
	if (res.status === 409) throw new ConflictError();
	if (!res.ok) throw new Error(`Failed to create source target: ${res.status}`);
	return res.json();
}

export interface UpdateSourceTargetPayload {
	enabled?: boolean;
	check_interval_minutes?: number;
}

export async function updateSourceTarget(
	id: string,
	patch: UpdateSourceTargetPayload,
): Promise<SourceTarget> {
	if (useMocks()) {
		const { updateSourceTarget: mockUpdate } = await import("../mocks/db");
		await mockDelay(80);
		return mockUpdate(id, patch);
	}
	return apiFetch<SourceTarget>(`/source-targets/${id}`, {
		method: "PATCH",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(patch),
	});
}

export async function deleteSourceTarget(id: string): Promise<void> {
	return apiFetchVoid(`/source-targets/${id}`, { method: "DELETE" });
}
