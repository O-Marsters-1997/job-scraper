import { createQuery, keepPreviousData } from "@tanstack/solid-query";
import { type Accessor, createEffect, createSignal, onCleanup } from "solid-js";
import { keys } from "../api/keys";
import { previewScoring } from "../api/scoringPreview";
import type { Preferences } from "../types/scoringConfig";

const PREVIEW_DEBOUNCE_MS = 400;

export function useScoringPreview(
	preferences: Accessor<Preferences>,
	enabled: Accessor<boolean>,
) {
	const [settled, setSettled] = createSignal(preferences());
	createEffect(() => {
		const next = preferences();
		const timer = setTimeout(() => setSettled(next), PREVIEW_DEBOUNCE_MS);
		onCleanup(() => clearTimeout(timer));
	});
	return createQuery(() => ({
		queryKey: keys.scoringPreview(settled()),
		queryFn: () => previewScoring(settled()),
		enabled: enabled(),
		placeholderData: keepPreviousData,
	}));
}
