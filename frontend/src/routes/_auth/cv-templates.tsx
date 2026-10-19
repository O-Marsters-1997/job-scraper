import { createFileRoute } from "@tanstack/solid-router";
import { LetterSkeletonEditor } from "@/components/cv/LetterSkeletonEditor";
import { cvTemplatesQueryOptions } from "../../hooks/useCVTemplates";
import { queryClient } from "../../lib/queryClient";
import { TrackedDocsSection } from "./-cv-templates/TrackedDocsSection";

export const Route = createFileRoute("/_auth/cv-templates")({
	loader: () =>
		Promise.all([
			queryClient.ensureQueryData(cvTemplatesQueryOptions("cv")),
			queryClient.ensureQueryData(cvTemplatesQueryOptions("letter")),
		]),
	component: CVTemplatesPage,
});

function CVTemplatesPage() {
	return (
		<div class="px-7 py-6">
			<TrackedDocsSection
				kind="cv"
				title="CVs"
				subtitle="Google Docs tracked as CVs"
			/>
			<TrackedDocsSection
				kind="letter"
				title="Cover Letters"
				subtitle="Google Docs holding your past cover letters"
			/>
			<section>
				<h3 class="text-base font-bold tracking-tight text-foreground">
					Letter skeleton
				</h3>
				<p class="mt-0.5 mb-4 text-xs text-faint">
					Your letter skeleton: intro, body section purposes and conclusion
				</p>
				<LetterSkeletonEditor />
			</section>
		</div>
	);
}
