import { createFileRoute } from "@tanstack/solid-router";
import { cvTemplatesQueryOptions } from "../../hooks/useCVTemplates";
import { queryClient } from "../../lib/queryClient";
import { TrackedDocsSection } from "./-cv-templates/TrackedDocsSection";

export const Route = createFileRoute("/_auth/cv-templates")({
	loader: () => queryClient.ensureQueryData(cvTemplatesQueryOptions("cv")),
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
		</div>
	);
}
