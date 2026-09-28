import { type Profile, profileSchema } from "../types/profile";
import { apiFetch, apiFetchVoid } from "./client";
import { mockDelay, mocked } from "./config";

export async function fetchProfile(): Promise<Profile> {
	return mocked(
		async (db) => {
			await mockDelay();
			return db.getProfile();
		},
		() => apiFetch("/profile", undefined, profileSchema),
	);
}

export async function updateProfile(payload: { email: string }): Promise<void> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			db.updateProfile(payload);
		},
		() =>
			apiFetchVoid("/profile", {
				method: "PUT",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify(payload),
			}),
	);
}
