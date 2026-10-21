import { Show } from "solid-js";
import { Icon } from "@/components/Icon";
import JobDescription from "@/components/jobs/JobDescription";
import { QueryBoundary } from "@/components/QueryBoundary";
import {
	Dialog,
	DialogClose,
	DialogContent,
	DialogTitle,
} from "@/components/ui/dialog";
import { useJob } from "@/hooks/useJobs";

interface JobPreviewDialogProps {
	jobId: string | null;
	onClose: () => void;
}

export function JobPreviewDialog(props: JobPreviewDialogProps) {
	return (
		<Dialog
			open={props.jobId !== null}
			onOpenChange={(open) => {
				if (!open) props.onClose();
			}}
		>
			<DialogContent class="flex max-h-[85vh] max-w-2xl flex-col">
				<Show when={props.jobId} keyed>
					{(id) => <Preview id={id} />}
				</Show>
			</DialogContent>
		</Dialog>
	);
}

function Preview(props: { id: string }) {
	const job = useJob(() => props.id);
	return (
		<QueryBoundary query={job} fallbackRows={6}>
			{(j) => (
				<>
					<div class="mb-4 flex items-start justify-between gap-3">
						<div class="flex min-w-0 flex-col gap-1">
							<DialogTitle>{j().Title}</DialogTitle>
							<span class="text-sm text-muted">{j().CompanySlug}</span>
						</div>
						<DialogClose aria-label="Close preview" class="shrink-0 text-muted">
							<Icon name="x" size={16} />
						</DialogClose>
					</div>
					<div class="min-h-0 flex-1 overflow-y-auto">
						<Show
							when={j().Description}
							fallback={
								<p class="text-sm text-muted">No description available.</p>
							}
						>
							{(html) => <JobDescription html={html()} />}
						</Show>
					</div>
				</>
			)}
		</QueryBoundary>
	);
}
