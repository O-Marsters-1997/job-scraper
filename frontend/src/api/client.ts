import { API_BASE } from "./config";

export async function apiFetch<T>(
	path: string,
	init?: RequestInit,
): Promise<T> {
	const res = await fetch(`${API_BASE}${path}`, {
		credentials: "include",
		...init,
	});
	if (!res.ok) throw new Error(`${path}: ${res.status}`);
	return res.json() as Promise<T>;
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
