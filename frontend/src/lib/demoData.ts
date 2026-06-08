import { createRoot, createSignal } from "solid-js";
import { queryClient } from "./queryClient";

export const DEMO_DATA_STORAGE_KEY = "job-scraper-demo-data";

function load(): boolean {
	try {
		return sessionStorage.getItem(DEMO_DATA_STORAGE_KEY) === "true";
	} catch {
		return false;
	}
}

// Module-level signal wrapped in createRoot to avoid reactive-owner warnings.
const [demoData, setDemoDataSignal] = createRoot(() => createSignal(load()));

// Reactive accessor — use in components: checked={demoDataEnabled()}
export { demoData as demoDataEnabled };

// Non-reactive read — safe to call inside async API functions.
export function isDemoData(): boolean {
	return demoData();
}

export function setDemoData(value: boolean): void {
	setDemoDataSignal(value);
	try {
		sessionStorage.setItem(DEMO_DATA_STORAGE_KEY, String(value));
	} catch {
		// Private/incognito mode may block sessionStorage writes — ignore.
	}
	// Query keys don't encode the flag; invalidate so active queries refetch.
	queryClient.invalidateQueries();
}
