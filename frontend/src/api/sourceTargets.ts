import type {
	CreateSourceTargetPayload,
	SourceTarget,
	UpdateSourceTargetPayload,
} from "../types/sourceTarget";
import { sourceTargetSchema } from "../types/sourceTarget";
import { apiFetch, apiFetchVoid } from "./client";
import { API_BASE, mockDelay, useMocks } from "./config";

export class ConflictError extends Error {
	constructor() {
		super("source target already exists");
	}
}

export async function fetchSourceTargets(): Promise<SourceTarget[]> {
	if (useMocks()) {
		const { getSourceTargets } = await import("../mocks/db");
		await mockDelay();
		return getSourceTargets();
	}
	return apiFetch("/source-targets", undefined, sourceTargetSchema.array());
}

export async function createSourceTarget(
	payload: CreateSourceTargetPayload,
): Promise<SourceTarget> {
	if (useMocks()) {
		const { createSourceTarget: mockCreate } = await import("../mocks/db");
		await mockDelay(80);
		const target = mockCreate(payload);
		if (!target) throw new ConflictError();
		return target;
	}
	const response = await fetch(`${API_BASE}/source-targets`, {
		method: "POST",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	});
	if (response.status === 409) throw new ConflictError();
	if (!response.ok)
		throw new Error(`Failed to create source target: ${response.status}`);
	return sourceTargetSchema.parse(await response.json());
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
	return apiFetch(
		`/source-targets/${id}`,
		{
			method: "PATCH",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify(patch),
		},
		sourceTargetSchema,
	);
}

export async function deleteSourceTarget(id: string): Promise<void> {
	if (useMocks()) {
		const { deleteSourceTarget: mockDelete } = await import("../mocks/db");
		await mockDelay(80);
		mockDelete(id);
		return;
	}
	return apiFetchVoid(`/source-targets/${id}`, { method: "DELETE" });
}

export async function rerunSourceTarget(id: string): Promise<SourceTarget> {
	if (useMocks()) {
		const { rerunSourceTarget: mockRerun } = await import("../mocks/db");
		await mockDelay(80);
		return mockRerun(id);
	}
	return apiFetch(
		`/source-targets/${id}/scrape`,
		{ method: "POST" },
		sourceTargetSchema,
	);
}
