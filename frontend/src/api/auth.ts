import { API_BASE, MOCK_BUILD } from "./config";

export function login(username: string, password: string) {
	if (MOCK_BUILD) {
		void username;
		void password;
		return Promise.resolve(
			new Response(JSON.stringify({ ok: true }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
	}
	return fetch(`${API_BASE}/auth/login`, {
		method: "POST",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ username, password }),
	});
}

export function signup(username: string, password: string) {
	if (MOCK_BUILD) {
		void username;
		void password;
		return Promise.resolve(
			new Response(JSON.stringify({ ok: true }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
	}
	return fetch(`${API_BASE}/auth/signup`, {
		method: "POST",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ username, password }),
	});
}

export function logout() {
	if (MOCK_BUILD) {
		return Promise.resolve(new Response(null, { status: 200 }));
	}
	return fetch(`${API_BASE}/auth/logout`, {
		method: "POST",
		credentials: "include",
	});
}

export function getMe() {
	if (MOCK_BUILD) {
		return Promise.resolve(
			new Response(JSON.stringify({ id: "user-1", username: "demo" }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
	}
	return fetch(`${API_BASE}/auth/me`, {
		credentials: "include",
	});
}
