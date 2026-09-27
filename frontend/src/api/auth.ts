import { type Me, meSchema } from "../types/me";
import { apiFetch } from "./client";
import { API_BASE, MOCK_BUILD, mockDelay } from "./config";

export async function login(
	username: string,
	password: string,
): Promise<Response> {
	if (MOCK_BUILD) {
		void password;
		const { switchMockUser } = await import("../mocks/db");
		switchMockUser(username);
		return new Response(JSON.stringify({ ok: true }), {
			status: 200,
			headers: { "Content-Type": "application/json" },
		});
	}
	return fetch(`${API_BASE}/auth/login`, {
		method: "POST",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ username, password }),
	});
}

export async function signup(
	username: string,
	password: string,
): Promise<Response> {
	if (MOCK_BUILD) {
		void password;
		const { switchMockUser } = await import("../mocks/db");
		switchMockUser(username);
		return new Response(JSON.stringify({ ok: true }), {
			status: 200,
			headers: { "Content-Type": "application/json" },
		});
	}
	return fetch(`${API_BASE}/auth/signup`, {
		method: "POST",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ username, password }),
	});
}

export function logout(): Promise<Response> {
	if (MOCK_BUILD) {
		return Promise.resolve(new Response(null, { status: 200 }));
	}
	return fetch(`${API_BASE}/auth/logout`, {
		method: "POST",
		credentials: "include",
	});
}

export async function fetchMe(): Promise<Me> {
	if (MOCK_BUILD) {
		const { mockUser } = await import("../mocks/db");
		await mockDelay();
		return mockUser;
	}
	return apiFetch("/auth/me", undefined, meSchema);
}
