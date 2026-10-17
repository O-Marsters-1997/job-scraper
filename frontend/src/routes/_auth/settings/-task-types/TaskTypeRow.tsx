import { Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { TaskType } from "@/types/task";
import { StatusColourPicker } from "../-statuses/StatusColourPicker";

export function TaskTypeRow(props: {
	type: TaskType;
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
							style={{ background: props.type.Colour }}
						/>
						<span class="flex-1 text-sm font-medium text-foreground">
							{props.type.Name}
						</span>
						<span class="text-xs text-muted-foreground">
							{props.type.TaskCount} task{props.type.TaskCount === 1 ? "" : "s"}
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
						aria-label="Task type name"
						value={props.name}
						onInput={(e) => props.onName(e.currentTarget.value)}
					/>
				</div>
				<Button
					type="submit"
					disabled={props.pending || !props.name.trim()}
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
