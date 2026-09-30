import type {
	CreateSourceTargetPayload,
	SourceTarget,
	UpdateSourceTargetPayload,
} from "../types/sourceTarget";
import { sourceTargetSchema } from "../types/sourceTarget";
import { apiFetch, apiFetchVoid, jsonInit, rethrowStatus } from "./client";
import { mocked } from "./config";

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
		(db) => db.getSourceTargets(),
		() => apiFetch("/source-targets", sourceTargetSchema.array()),
	);
}

export async function createSourceTarget(
	payload: CreateSourceTargetPayload,
): Promise<SourceTarget> {
	return mocked(
		(db) => {
			const target = db.createSourceTarget(payload);
			if (!target) throw new ConflictError();
			return target;
		},
		() =>
			apiFetch(
				"/source-targets",
				sourceTargetSchema,
				jsonInit("POST", payload),
			).catch(rethrowStatus({ 409: () => new ConflictError() })),
	);
}

export async function updateSourceTarget(
	id: string,
	patch: UpdateSourceTargetPayload,
): Promise<SourceTarget> {
	return mocked(
		(db) => db.updateSourceTarget(id, patch),
		() =>
			apiFetch(
				`/source-targets/${id}`,
				sourceTargetSchema,
				jsonInit("PATCH", patch),
			),
	);
}

export async function deleteSourceTarget(
	id: string,
	opts?: { keepalive?: boolean },
): Promise<void> {
	return mocked(
		(db) => db.deleteSourceTarget(id),
		() =>
			apiFetchVoid(`/source-targets/${id}`, {
				method: "DELETE",
				keepalive: opts?.keepalive ?? false,
			}),
	);
}

export async function rerunSourceTarget(id: string): Promise<SourceTarget> {
	return mocked(
		(db) => db.rerunSourceTarget(id),
		() =>
			apiFetch(`/source-targets/${id}/scrape`, sourceTargetSchema, {
				method: "POST",
			}).catch(rethrowStatus({ 409: () => new AlreadyRunningError() })),
	);
}
