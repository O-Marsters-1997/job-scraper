import { type Profile, profileSchema } from "../types/profile";
import { apiFetch, apiFetchVoid } from "./client";
import { mockDelay, useMocks } from "./config";

export async function fetchProfile(): Promise<Profile> {
	if (useMocks()) {
		const { getProfile } = await import("../mocks/db");
		await mockDelay();
		return getProfile();
	}
	return apiFetch("/profile", undefined, profileSchema);
}

export async function updateProfile(payload: { email: string }): Promise<void> {
	if (useMocks()) {
		const { updateProfile: setMockProfile } = await import("../mocks/db");
		await mockDelay(80);
		setMockProfile(payload);
		return;
	}
	return apiFetchVoid("/profile", {
		method: "PUT",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	});
}
