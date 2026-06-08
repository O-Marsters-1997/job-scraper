export const API_BASE = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

// Mocks are ON by default on this branch. Set VITE_MOCK=false to hit the real API.
export const USE_MOCKS = import.meta.env.VITE_MOCK !== "false";

export const mockDelay = (ms = 150): Promise<void> =>
	new Promise((resolve) => setTimeout(resolve, ms));
