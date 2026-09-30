import { demoDataEnabled } from "../lib/demoData";

export const API_BASE = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export const MOCK_BUILD = import.meta.env.VITE_MOCK === "true";

const MOCK_DELAY_MS = 150;

type MocksModule = typeof import("../mocks/db");

// The db import stays inside this function, not at each caller's top level,
// so mocks/db.ts is only ever pulled into the mock build's bundle.
export async function mocked<T>(
	mock: (db: MocksModule) => T | Promise<T>,
	real: () => T | Promise<T>,
): Promise<T> {
	if (MOCK_BUILD || demoDataEnabled()) {
		const db = await import("../mocks/db");
		await new Promise((resolve) => setTimeout(resolve, MOCK_DELAY_MS));
		return mock(db);
	}
	return real();
}
