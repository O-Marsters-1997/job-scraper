import { apiFetch, apiFetchVoid } from "./client";

export interface Profile {
	username: string;
	email: string;
}

export async function fetchProfile(): Promise<Profile> {
	return apiFetch<Profile>("/profile");
}

export async function updateProfile(payload: { email: string }): Promise<void> {
	return apiFetchVoid("/profile", {
		method: "PUT",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	});
}
