import type {
	CreateSourceTargetPayload,
	SourceTarget,
	UpdateSourceTargetPayload,
} from "../types/sourceTarget";
import { sourceTargetSchema } from "../types/sourceTarget";
import { apiFetch, apiFetchVoid } from "./client";
import { API_BASE, mockDelay, mocked } from "./config";

export class ConflictError extends Error {
	constructor() {
		super("source target already exists");
	}
}

export class AlreadyRunningError extends Error {
	constructor() {
		super("source target is already running");
	}
}

export async function fetchSourceTargets(): Promise<SourceTarget[]> {
	return mocked(
		async (db) => {
			await mockDelay();
			return db.getSourceTargets();
		},
		() => apiFetch("/source-targets", undefined, sourceTargetSchema.array()),
	);
}

export async function createSourceTarget(
	payload: CreateSourceTargetPayload,
): Promise<SourceTarget> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			const target = db.createSourceTarget(payload);
			if (!target) throw new ConflictError();
			return target;
		},
		async () => {
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
		},
	);
}

export async function updateSourceTarget(
	id: string,
	patch: UpdateSourceTargetPayload,
): Promise<SourceTarget> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			return db.updateSourceTarget(id, patch);
		},
		() =>
			apiFetch(
				`/source-targets/${id}`,
				{
					method: "PATCH",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify(patch),
				},
				sourceTargetSchema,
			),
	);
}

export async function deleteSourceTarget(
	id: string,
	opts?: { keepalive?: boolean },
): Promise<void> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			db.deleteSourceTarget(id);
		},
		() =>
			apiFetchVoid(`/source-targets/${id}`, {
				method: "DELETE",
				keepalive: opts?.keepalive ?? false,
			}),
	);
}

export async function rerunSourceTarget(id: string): Promise<SourceTarget> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			return db.rerunSourceTarget(id);
		},
		async () => {
			const response = await fetch(`${API_BASE}/source-targets/${id}/scrape`, {
				method: "POST",
				credentials: "include",
			});
			if (response.status === 409) throw new AlreadyRunningError();
			if (!response.ok)
				throw new Error(`Failed to start search: ${response.status}`);
			return sourceTargetSchema.parse(await response.json());
		},
	);
}
