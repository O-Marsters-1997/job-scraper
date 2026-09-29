import { Show } from "solid-js";
import { Input } from "@/components/ui/input";
import type { ApplicationStatus } from "@/types/applicationStatus";
import { StatusColourPicker } from "./StatusColourPicker";

const ghostButton =
	"rounded px-2 py-1 text-xs font-medium text-muted transition hover:bg-surface-muted hover:text-foreground";

export function StatusRow(props: {
	status: ApplicationStatus;
	editing: boolean;
	name: string;
	colour: string;
	pending: boolean;
	onName: (name: string) => void;
	onColour: (hex: string) => void;
	onEdit: () => void;
	onCancel: () => void;
	onDelete: () => void;
	onSubmit: (e: SubmitEvent) => void;
}) {
	return (
		<form
			onSubmit={(e) => props.onSubmit(e)}
			class="flex items-center gap-3 px-4 py-3"
		>
			<Show
				when={props.editing}
				fallback={
					<>
						<span
							class="size-3 shrink-0 rounded-full"
							style={{ background: props.status.Colour }}
						/>
						<span class="flex-1 text-sm font-medium text-foreground">
							{props.status.Name}
						</span>
						<button
							type="button"
							onClick={() => props.onEdit()}
							class={ghostButton}
						>
							Edit
						</button>
						<button
							type="button"
							onClick={() => props.onDelete()}
							class="rounded px-2 py-1 text-xs font-medium text-destructive-strong transition hover:bg-destructive-subtle"
						>
							Delete
						</button>
					</>
				}
			>
				<div class="flex flex-1 items-center gap-2">
					<StatusColourPicker value={props.colour} onChange={props.onColour} />
					<Input
						class="flex-1"
						aria-label="Status name"
						value={props.name}
						onInput={(e) => props.onName(e.currentTarget.value)}
					/>
				</div>
				<button
					type="submit"
					disabled={props.pending}
					class="rounded px-2 py-1 text-xs font-medium text-primary transition hover:bg-accent-subtle disabled:opacity-50"
				>
					Save
				</button>
				<button
					type="button"
					onClick={() => props.onCancel()}
					class={ghostButton}
				>
					Cancel
				</button>
			</Show>
		</form>
	);
}
