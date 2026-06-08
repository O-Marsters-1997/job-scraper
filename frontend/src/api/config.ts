import { isDemoData } from "../lib/demoData";

export const API_BASE = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

// Build-time full-mock mode for local dev with no backend. Set VITE_MOCK=true to enable.
export const MOCK_BUILD = import.meta.env.VITE_MOCK === "true";

// Data modules should mock when build-mock is on OR the user enabled demo data this session.
export function useMocks(): boolean {
	return MOCK_BUILD || isDemoData();
}

export const mockDelay = (ms = 150): Promise<void> =>
	new Promise((resolve) => setTimeout(resolve, ms));
