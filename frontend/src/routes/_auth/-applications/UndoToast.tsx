import type { JSX } from "solid-js";
import { Kbd } from "@/components/Kbd";

export function UndoToast(props: {
	message: string;
	onUndo: () => void;
}): JSX.Element {
	return (
		<output class="fixed bottom-4 left-1/2 z-50 flex -translate-x-1/2 items-center gap-3 rounded-lg border border-border bg-surface px-4 py-2 text-sm shadow-xl">
			<span>{props.message}</span>
			<button
				type="button"
				onClick={() => props.onUndo()}
				class="flex items-center gap-1.5 font-medium text-primary hover:underline"
			>
				Undo <Kbd>z</Kbd>
			</button>
		</output>
	);
}
