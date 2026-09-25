import { isDemoData } from "../lib/demoData";

export const API_BASE = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export const MOCK_BUILD = import.meta.env.VITE_MOCK === "true";

export function useMocks(): boolean {
	return MOCK_BUILD || isDemoData();
}

export const mockDelay = (ms = 150): Promise<void> =>
	new Promise((resolve) => setTimeout(resolve, ms));
