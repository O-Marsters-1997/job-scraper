import type { AiPrefs } from "../types/aiPrefs";
import type { GoogleStatus } from "../types/google";
import type { Profile } from "../types/profile";
import type { Quota } from "../types/quota";

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

const disconnectedGoogle: GoogleStatus = {
	connected: false,
	canWrite: false,
	canEditDocs: false,
};
let googleStatus: GoogleStatus = {
	connected: true,
	canWrite: true,
	canEditDocs: true,
};

export function getGoogleStatus(): GoogleStatus {
	return structuredClone(googleStatus);
}

export function disconnectGoogle(): void {
	googleStatus = disconnectedGoogle;
}

let profile: Profile = { username: "demo", email: "demo@example.com" };

export function getProfile(): Profile {
	return structuredClone(profile);
}

export function updateProfile(payload: { email: string }): void {
	profile = { ...profile, email: payload.email };
}

export function getAiUsage(): Quota {
	const base = {
		provider: "openrouter",
		unit: "USD",
		resetsAt: null,
		fetchedAt: new Date().toISOString(),
		error: "",
	};
	if (!aiPrefs.configuredProviders.includes("openrouter")) {
		return {
			...base,
			status: "not_configured",
			used: null,
			limit: null,
			percent: null,
			level: "ok",
			fetchedAt: null,
		};
	}
	return {
		...base,
		status: "ok",
		used: 3.2,
		limit: 10,
		percent: 32,
		level: "ok",
		resetsAt: new Date(Date.UTC(2026, 10, 1)).toISOString(),
	};
}
