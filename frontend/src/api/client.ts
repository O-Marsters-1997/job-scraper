import type { z } from "zod";
import { API_BASE } from "./config";

export async function apiFetch<S extends z.ZodType>(
	path: string,
	init: RequestInit | undefined,
	schema: S,
): Promise<z.infer<S>> {
	const res = await fetch(`${API_BASE}${path}`, {
		credentials: "include",
		...init,
	});
	if (!res.ok) throw new Error(`${path}: ${res.status}`);
	const data: unknown = await res.json();
	return schema.parse(data);
}

export async function apiFetchVoid(
	path: string,
	init?: RequestInit,
): Promise<void> {
	const res = await fetch(`${API_BASE}${path}`, {
		credentials: "include",
		...init,
	});
	if (!res.ok) throw new Error(`${path}: ${res.status}`);
}

export async function apiFetchBlob(
	path: string,
	init?: RequestInit,
): Promise<ArrayBuffer> {
	const res = await fetch(`${API_BASE}${path}`, {
		credentials: "include",
		...init,
	});
	if (!res.ok) throw new Error(`${path}: ${res.status}`);
	return res.arrayBuffer();
}
