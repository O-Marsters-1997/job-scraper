import { createSignal, onCleanup } from "solid-js";
import { createStore, produce } from "solid-js/store";
import { streamSuggestion } from "@/api/tailoring";
import type { DraftFinding, SuggestAction } from "@/types/tailoring";

export type SlotSuggestion = {
	action: SuggestAction;
	prompt: string;
	before: string;
	text: string;
	done: boolean;
	findings: DraftFinding[];
	error: string | undefined;
};

export function createSuggestions(opts: {
	draftId: string;
	text: (slotId: string) => string;
	maxChars: (slotId: string) => number;
	apply: (slotId: string, text: string) => void;
}) {
	const [items, setItems] = createStore<Record<string, SlotSuggestion>>({});
	const [announcement, setAnnouncement] = createSignal("");
	const controllers = new Map<string, AbortController>();

	const drop = (slotId: string) => {
		controllers.get(slotId)?.abort();
		controllers.delete(slotId);
		setItems(
			produce((all) => {
				delete all[slotId];
			}),
		);
	};

	const ask = async (slotId: string, action: SuggestAction, prompt = "") => {
		controllers.get(slotId)?.abort();
		const ctrl = new AbortController();
		controllers.set(slotId, ctrl);
		const before = opts.text(slotId);
		setAnnouncement("");
		setItems(slotId, {
			action,
			prompt,
			before,
			text: "",
			done: false,
			findings: [],
			error: undefined,
		});
		try {
			const result = await streamSuggestion(
				opts.draftId,
				slotId,
				{ action, prompt, text: before, maxChars: opts.maxChars(slotId) },
				(delta) => setItems(slotId, "text", (t) => t + delta),
				ctrl.signal,
			);
			if (ctrl.signal.aborted) return;
			setItems(slotId, {
				text: result.text,
				findings: result.findings,
				done: true,
			});
			setAnnouncement("Suggestion ready");
		} catch (err) {
			if (ctrl.signal.aborted) return;
			setItems(slotId, {
				done: true,
				error: err instanceof Error ? err.message : "Suggestion failed",
			});
		}
	};

	onCleanup(() => {
		for (const c of controllers.values()) c.abort();
	});

	return {
		get: (slotId: string): SlotSuggestion | undefined => items[slotId],
		announcement,
		ask,
		retry: (slotId: string) => {
			const s = items[slotId];
			if (s) void ask(slotId, s.action, s.prompt);
		},
		accept: (slotId: string) => {
			const s = items[slotId];
			if (!s?.done || s.error !== undefined) return;
			opts.apply(slotId, s.text);
			drop(slotId);
		},
		reject: drop,
	};
}

export type SuggestionsState = ReturnType<typeof createSuggestions>;
