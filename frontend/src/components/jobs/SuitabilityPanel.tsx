import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Job } from "@/types/job";
import { CriteriaBreakdown } from "./CriteriaBreakdown";
import { SuitabilityScoreValue } from "./SuitabilityScoreValue";

export function SuitabilityPanel(props: { job: Job }) {
	return (
		<Card>
			<CardHeader class="pb-2">
				<CardTitle>Suitability</CardTitle>
			</CardHeader>
			<CardContent class="gap-3">
				<SuitabilityScoreValue
					score={props.job.SuitabilityScore}
					confidence={props.job.Confidence}
					size="lg"
				/>
				<CriteriaBreakdown job={props.job} />
			</CardContent>
		</Card>
	);
}
