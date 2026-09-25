import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Job } from "@/types/job";
import { SuitabilityScoreValue } from "./SuitabilityScoreValue";

export function SuitabilityPanel(props: { job: Job }) {
	return (
		<Card>
			<CardHeader class="pb-2">
				<CardTitle>Suitability</CardTitle>
			</CardHeader>
			<CardContent class="gap-3">
				<SuitabilityScoreValue score={props.job.SuitabilityScore} size="lg" />
			</CardContent>
		</Card>
	);
}
