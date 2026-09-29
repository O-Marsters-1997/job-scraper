import type { AiPrefs } from "../types/aiPrefs";
import type { GoogleStatus } from "../types/google";
import type { Profile } from "../types/profile";

let aiPrefs: AiPrefs = {
	configuredProviders: ["openrouter"],
	scoringEnabled: true,
};

export function getAiPrefs(): AiPrefs {
	return structuredClone(aiPrefs);
}

export function setAiCredential(provider: string, apiKey: string | null): void {
	const providers = new Set(aiPrefs.configuredProviders);
	if (apiKey) providers.add(provider);
	else providers.delete(provider);
	aiPrefs = { ...aiPrefs, configuredProviders: [...providers] };
}

let googleStatus: GoogleStatus = { connected: false, canWrite: false };

export function getGoogleStatus(): GoogleStatus {
	return structuredClone(googleStatus);
}

export function disconnectGoogle(): void {
	googleStatus = { connected: false, canWrite: false };
}

let profile: Profile = { username: "demo", email: "demo@example.com" };

export function getProfile(): Profile {
	return structuredClone(profile);
}

export function updateProfile(payload: { email: string }): void {
	profile = { ...profile, email: payload.email };
}
