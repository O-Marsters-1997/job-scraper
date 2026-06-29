import type { ZodSchema } from "zod";
import { API_BASE } from "./config";

export async function apiFetch<T>(
	path: string,
	init?: RequestInit,
	schema?: ZodSchema<T>,
): Promise<T> {
	const res = await fetch(`${API_BASE}${path}`, {
		credentials: "include",
		...init,
	});
	if (!res.ok) throw new Error(`${path}: ${res.status}`);
	const data: unknown = await res.json();
	return schema ? schema.parse(data) : (data as T);
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
