import { Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { ApplicationStatus } from "@/types/applicationStatus";
import { StatusColourPicker } from "./StatusColourPicker";

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
						<Button
							type="button"
							onClick={() => props.onEdit()}
							variant="ghost"
							size="sm"
							class="h-auto px-2 py-1 text-xs"
						>
							Edit
						</Button>
						<Button
							type="button"
							onClick={() => props.onDelete()}
							variant="ghost"
							size="sm"
							class="h-auto px-2 py-1 text-xs text-destructive-strong hover:bg-destructive-subtle hover:text-destructive-strong"
						>
							Delete
						</Button>
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
				<Button
					type="submit"
					disabled={props.pending}
					variant="ghost"
					size="sm"
					class="h-auto px-2 py-1 text-xs text-primary hover:text-primary"
				>
					Save
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
			</Show>
		</form>
	);
}
