import { type Me, meSchema } from "../types/me";
import { apiFetch, apiFetchVoid, jsonInit } from "./client";
import { MOCK_BUILD, mocked } from "./config";

export async function auth(
	path: "/auth/login" | "/auth/signup",
	username: string,
	password: string,
): Promise<void> {
	if (MOCK_BUILD) {
		const { switchMockUser } = await import("../mocks/db");
		switchMockUser(username);
		return;
	}
	await apiFetchVoid(path, jsonInit("POST", { username, password }));
}

export async function logout(): Promise<void> {
	if (MOCK_BUILD) return;
	await apiFetchVoid("/auth/logout", { method: "POST" });
}

export async function fetchMe(): Promise<Me> {
	return mocked(
		(db) => db.mockUser,
		() => apiFetch("/auth/me", meSchema),
	);
}
