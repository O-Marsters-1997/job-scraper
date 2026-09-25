import type { Job } from "@/types/job";
import { SuitabilityScoreValue } from "./SuitabilityScoreValue";

interface Props {
	job: Job;
}

export function JobRowExpander(props: Props) {
	return (
		<div class="flex flex-col gap-2.5 px-4 py-3 bg-surface-muted border-t border-border">
			<SuitabilityScoreValue score={props.job.SuitabilityScore} size="sm" />
		</div>
	);
}
