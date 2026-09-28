import { isDemoData } from "../lib/demoData";

export const API_BASE = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export const MOCK_BUILD = import.meta.env.VITE_MOCK === "true";

export function useMocks(): boolean {
	return MOCK_BUILD || isDemoData();
}

export const mockDelay = (ms = 150): Promise<void> =>
	new Promise((resolve) => setTimeout(resolve, ms));

type MocksModule = typeof import("../mocks/db");

// The db import stays inside this function, not at each caller's top level,
// so mocks/db.ts is only ever pulled into the mock build's bundle.
export async function mocked<T>(
	mock: (db: MocksModule) => T | Promise<T>,
	real: () => T | Promise<T>,
): Promise<T> {
	if (useMocks()) {
		const db = await import("../mocks/db");
		return mock(db);
	}
	return real();
}
