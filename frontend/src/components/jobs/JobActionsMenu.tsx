import { Show } from "solid-js";
import { Icon } from "@/components/Icon";
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
}

export function JobActionsMenu(props: JobActionsMenuProps) {
	return (
		<DropdownMenu>
			<DropdownMenuTrigger
				as="button"
				class="flex h-8 w-8 items-center justify-center rounded-md text-faint transition-colors hover:bg-accent-subtle hover:text-foreground"
				aria-label="Job actions"
			>
				<Icon name="moreVertical" />
			</DropdownMenuTrigger>
			<DropdownMenuContent>
				<DropdownMenuItem
					as="a"
					href={props.job.URL}
					target="_blank"
					rel="noopener noreferrer"
					class="cursor-pointer"
				>
					<Icon name="externalLink" size={14} />
					Apply
				</DropdownMenuItem>
				<DropdownMenuSeparator />
				<Show
					when={props.appSummary}
					fallback={
						<DropdownMenuItem onSelect={props.onTrack}>
							<Icon name="plus" size={14} />
							Track application
						</DropdownMenuItem>
					}
				>
					{(summary) => (
						<DropdownMenuItem onSelect={props.onTrack}>
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
