import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Job } from "@/types/job";
import { ExperienceMatch } from "./ExperienceMatch";
import { GradeControl } from "./GradeControl";
import { ScoreBreakdown } from "./ScoreBreakdown";
import { SuitabilityScoreValue } from "./SuitabilityScoreValue";

export function SuitabilityPanel(props: { job: Job }) {
	return (
		<Card
			data-feedback-job={
				props.job.SuitabilityScore == null ? undefined : props.job.ID
			}
		>
			<CardHeader class="pb-2">
				<CardTitle>Suitability</CardTitle>
			</CardHeader>
			<CardContent class="gap-3">
				<SuitabilityScoreValue
					score={props.job.SuitabilityScore}
					band={props.job.Band}
					breakdown={props.job.Breakdown}
					size="lg"
				/>
				<ScoreBreakdown job={props.job} />
				<ExperienceMatch jobId={props.job.ID} />
				<GradeControl jobId={props.job.ID} band={props.job.Band} />
			</CardContent>
		</Card>
	);
}
