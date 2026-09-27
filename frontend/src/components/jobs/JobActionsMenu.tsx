import { Show } from "solid-js";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { STATUS_FALLBACK_COLOUR } from "@/lib/status";
import type { JobApplicationSummary } from "@/types/application";
import type { Job } from "@/types/job";

interface JobActionsMenuProps {
	job: Job;
	appSummary: JobApplicationSummary | undefined;
	onTrack: () => void;
	onEdit: () => void;
}

export function JobActionsMenu(props: JobActionsMenuProps) {
	return (
		<DropdownMenu>
			<DropdownMenuTrigger
				as="button"
				class="flex h-8 w-8 items-center justify-center rounded-md text-faint transition-colors hover:bg-accent-subtle hover:text-foreground"
				aria-label="Job actions"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					width="16"
					height="16"
					viewBox="0 0 24 24"
					fill="currentColor"
					aria-hidden="true"
				>
					<circle cx="12" cy="5" r="1.5" />
					<circle cx="12" cy="12" r="1.5" />
					<circle cx="12" cy="19" r="1.5" />
				</svg>
			</DropdownMenuTrigger>
			<DropdownMenuContent>
				<DropdownMenuItem
					as="a"
					href={props.job.URL}
					target="_blank"
					rel="noopener noreferrer"
					class="cursor-pointer"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="14"
						height="14"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						aria-hidden="true"
					>
						<path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
						<polyline points="15 3 21 3 21 9" />
						<line x1="10" y1="14" x2="21" y2="3" />
					</svg>
					Apply
				</DropdownMenuItem>
				<DropdownMenuSeparator />
				<Show
					when={props.appSummary}
					fallback={
						<DropdownMenuItem onSelect={props.onTrack}>
							<svg
								xmlns="http://www.w3.org/2000/svg"
								width="14"
								height="14"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
								aria-hidden="true"
							>
								<path d="M12 5v14M5 12h14" />
							</svg>
							Track application
						</DropdownMenuItem>
					}
				>
					{(summary) => (
						<DropdownMenuItem onSelect={props.onEdit}>
							<span
								class="size-2 shrink-0 rounded-full"
								style={{
									background: summary().StatusColour || STATUS_FALLBACK_COLOUR,
								}}
							/>
							{summary().StatusName || "Edit status"}
						</DropdownMenuItem>
					)}
				</Show>
			</DropdownMenuContent>
		</DropdownMenu>
	);
}
