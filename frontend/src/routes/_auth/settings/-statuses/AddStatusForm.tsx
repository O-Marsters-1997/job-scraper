import { Input } from "@/components/ui/input";
import { StatusColourPicker } from "./StatusColourPicker";

export function AddStatusForm(props: {
	name: string;
	colour: string;
	pending: boolean;
	onName: (name: string) => void;
	onColour: (hex: string) => void;
	onCancel: () => void;
	onSubmit: (e: SubmitEvent) => void;
}) {
	return (
		<form
			onSubmit={(e) => props.onSubmit(e)}
			class="flex items-center gap-2 px-4 py-3"
		>
			<StatusColourPicker value={props.colour} onChange={props.onColour} />
			<Input
				class="flex-1"
				aria-label="Status name"
				placeholder="Status name"
				value={props.name}
				onInput={(e) => props.onName(e.currentTarget.value)}
			/>
			<button
				type="submit"
				disabled={props.pending || !props.name.trim()}
				class="rounded px-2 py-1 text-xs font-medium text-primary transition hover:bg-accent-subtle disabled:opacity-50"
			>
				Add
			</button>
			<button
				type="button"
				onClick={() => props.onCancel()}
				class="rounded px-2 py-1 text-xs font-medium text-muted transition hover:bg-surface-muted hover:text-foreground"
			>
				Cancel
			</button>
		</form>
	);
}
