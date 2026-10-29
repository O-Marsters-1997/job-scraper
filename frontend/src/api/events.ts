import type { EventInput, ImpressionBatchInput } from "../types/event";
import { apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

// Fire and forget: a failed event never reaches the caller or the page.
export function recordEvent(input: EventInput): void {
	void mocked(
		() => undefined,
		() => apiFetchVoid("/events", jsonInit("POST", input)),
	).catch(() => undefined);
}

export function recordImpressions(input: ImpressionBatchInput): void {
	void mocked(
		() => undefined,
		() => apiFetchVoid("/events/impressions", jsonInit("POST", input)),
	).catch(() => undefined);
}
