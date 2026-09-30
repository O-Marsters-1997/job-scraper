import { createRoot, createSignal } from "solid-js";
import { queryClient } from "./queryClient";

const DEMO_DATA_STORAGE_KEY = "job-scraper-demo-data";

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

export function setDemoData(value: boolean): void {
	setDemoDataSignal(value);
	try {
		sessionStorage.setItem(DEMO_DATA_STORAGE_KEY, String(value));
	} catch {
		// Private/incognito mode may block sessionStorage writes — ignore.
	}
	queryClient.invalidateQueries();
}
