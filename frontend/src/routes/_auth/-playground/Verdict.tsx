import { createEffect, createSignal, For, onCleanup, onMount } from "solid-js";
import { Kbd } from "@/components/Kbd";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import type { FeedbackDirection } from "../../../types/scoreFeedback";

const VERDICTS: { key: string; label: string; direction: FeedbackDirection }[] =
	[
		{ key: "1", label: "Looks right", direction: "accurate" },
		{ key: "2", label: "Too high", direction: "lower" },
		{ key: "3", label: "Too low", direction: "higher" },
	];

function isTyping(el: EventTarget | null) {
	return (
		el instanceof HTMLElement &&
		(el.isContentEditable ||
			["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName))
	);
}

export function Verdict(props: {
	failed: boolean;
	onVerdict: (direction: FeedbackDirection, overall: string) => void;
}) {
	const [overall, setOverall] = createSignal("");
	const [sent, setSent] = createSignal<FeedbackDirection>();
	createEffect(() => {
		if (props.failed) setSent(undefined);
	});
	const submit = (direction: FeedbackDirection) => {
		if (sent()) return;
		setSent(direction);
		props.onVerdict(direction, overall());
	};
	const onKey = (e: KeyboardEvent) => {
		if (e.repeat || e.metaKey || e.ctrlKey || e.altKey || isTyping(e.target))
			return;
		const v = VERDICTS.find((x) => x.key === e.key);
		if (v) submit(v.direction);
	};
	onMount(() => window.addEventListener("keydown", onKey));
	onCleanup(() => window.removeEventListener("keydown", onKey));
	return (
		<div class="flex flex-col gap-3 border-t border-border pt-4">
			<Textarea
				rows={2}
				placeholder="Overall reason (optional)"
				aria-label="Overall reason"
				value={overall()}
				onInput={(e) => setOverall(e.currentTarget.value)}
			/>
			<div class="flex gap-2">
				<For each={VERDICTS}>
					{(v) => (
						<Button
							variant="outline"
							class="flex-1"
							disabled={!!sent()}
							loading={sent() === v.direction}
							onClick={() => submit(v.direction)}
						>
							<Kbd>{v.key}</Kbd>
							{v.label}
						</Button>
					)}
				</For>
			</div>
		</div>
	);
}
