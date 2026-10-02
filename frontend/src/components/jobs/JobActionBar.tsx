import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import { useApplicationStatuses } from "@/hooks/useApplicationStatuses";
import {
	useCreateApplication,
	useUpdateApplication,
} from "@/hooks/useApplications";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { dayKey } from "@/lib/datetime";
import type { ApplicationWithDetails } from "@/types/application";
import type { Job } from "@/types/job";

interface JobActionBarProps {
	job: Pick<Job, "ID" | "URL">;
	app: ApplicationWithDetails | undefined;
	onCv: () => void;
	onTrack: () => void;
}

export function JobActionBar(props: JobActionBarProps) {
	const statuses = useApplicationStatuses();
	const create = useCreateApplication();
	const update = useUpdateApplication();

	const applied = useFormSubmit(async () => {
		const status = statuses.data?.find(
			(s) => s.Name.trim().toLowerCase() === "applied",
		);
		if (!status) {
			props.onTrack();
			return;
		}
		const applied_at = dayKey(new Date());
		const app = props.app;
		if (!app) {
			await create.mutateAsync({
				job_id: props.job.ID,
				status_id: status.ID,
				applied_at,
			});
			return;
		}
		await update.mutateAsync({
			id: app.ID,
			data: {
				status_id: status.ID,
				applied_at,
				notes: app.Notes,
				salary_info: app.SalaryInfo,
			},
		});
	});

	return (
		<div class="fixed inset-x-0 bottom-0 z-30 border-t border-border bg-surface px-4 pt-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] md:hidden">
			<FormFeedback error={applied.error()} />
			<div class="grid grid-cols-3 gap-2">
				<Button variant="outline" onClick={props.onCv}>
					CV
				</Button>
				<Button
					as="a"
					variant="outline"
					href={props.job.URL}
					target="_blank"
					rel="noopener noreferrer"
				>
					Open listing
				</Button>
				<Button
					disabled={!statuses.data || applied.pending()}
					onClick={applied.submit}
				>
					I applied
				</Button>
			</div>
		</div>
	);
}
