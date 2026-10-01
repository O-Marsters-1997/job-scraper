import { Button } from "@/components/ui/button";
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
			<Button
				type="submit"
				disabled={props.pending || !props.name.trim()}
				variant="ghost"
				size="sm"
				class="h-auto px-2 py-1 text-xs text-primary"
			>
				Add
			</Button>
			<Button
				type="button"
				onClick={() => props.onCancel()}
				variant="ghost"
				size="sm"
				class="h-auto px-2 py-1 text-xs"
			>
				Cancel
			</Button>
		</form>
	);
}
