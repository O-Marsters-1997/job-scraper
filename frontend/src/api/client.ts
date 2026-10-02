import type { z } from "zod";
import { API_BASE } from "./config";

export class ApiError extends Error {
	constructor(
		readonly status: number,
		path: string,
		readonly body: unknown,
	) {
		super(`${path}: ${status}`);
	}
}

export const jsonInit = (method: string, body: unknown): RequestInit => ({
	method,
	headers: { "Content-Type": "application/json" },
	body: JSON.stringify(body),
});

export async function request(
	path: string,
	init?: RequestInit,
): Promise<Response> {
	const res = await fetch(`${API_BASE}${path}`, {
		credentials: "include",
		...init,
	});
	if (!res.ok) {
		throw new ApiError(
			res.status,
			path,
			await res.json().catch(() => undefined),
		);
	}
	return res;
}

export async function apiFetch<S extends z.ZodType>(
	path: string,
	schema: S,
	init?: RequestInit,
): Promise<z.infer<S>> {
	const res = await request(path, init);
	return schema.parse(await res.json());
}

export async function apiFetchVoid(
	path: string,
	init?: RequestInit,
): Promise<void> {
	await request(path, init);
}

export async function apiFetchBlob(
	path: string,
	init?: RequestInit,
): Promise<ArrayBuffer> {
	const res = await request(path, init);
	return res.arrayBuffer();
}

export const rethrowStatus =
	(errors: Record<number, (err: ApiError) => Error>) =>
	(err: unknown): never => {
		const make = err instanceof ApiError && errors[err.status];
		throw make ? make(err) : err;
	};
