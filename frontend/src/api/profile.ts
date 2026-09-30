import { type Profile, profileSchema } from "../types/profile";
import { apiFetch, apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchProfile(): Promise<Profile> {
	return mocked(
		(db) => db.getProfile(),
		() => apiFetch("/profile", profileSchema),
	);
}

export async function updateProfile(payload: { email: string }): Promise<void> {
	return mocked(
		(db) => db.updateProfile(payload),
		() => apiFetchVoid("/profile", jsonInit("PUT", payload)),
	);
}
