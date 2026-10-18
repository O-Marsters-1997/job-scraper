import { createFileRoute, Link } from "@tanstack/solid-router";
import { Icon } from "@/components/Icon";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { StatusBadge } from "@/components/StatusBadge";
import { Button } from "@/components/ui/button";
import {
	applicationsQueryOptions,
	useApplications,
} from "@/hooks/useApplications";
import { queryClient } from "@/lib/queryClient";
import { JobNotesCard } from "./-jobs-detail/JobNotesCard";

export const Route = createFileRoute("/_auth/applications_/$id")({
	loader: () => queryClient.prefetchQuery(applicationsQueryOptions()),
	component: ApplicationDetailPage,
});

function ApplicationDetailPage() {
	const params = Route.useParams();
	const query = useApplications();

	return (
		<QueryBoundary
			query={query}
			select={(apps) => apps.find((a) => a.ID === params().id)}
			notFound={
				<div class="flex h-[calc(100vh-14rem)] flex-col items-center justify-center gap-4 text-center">
					<Icon name="zoomIn" size={40} strokeWidth={1.5} class="text-faint" />
					<p class="text-base font-semibold text-foreground">
						Application not found
					</p>
					<Button as={Link} to="/applications" variant="outline" size="sm">
						← Back to Applications
					</Button>
				</div>
			}
		>
			{(app) => (
				<div class="px-4 py-6 md:px-7">
					<PageHeading
						title={app().JobTitle}
						subtitle={`${app().JobCompanySlug}${app().JobLocation ? ` · ${app().JobLocation}` : ""}`}
					>
						<div class="flex shrink-0 items-center gap-3">
							<StatusBadge
								name={app().StatusName}
								colour={app().StatusColour}
							/>
							<Link
								to="/jobs/$id"
								params={{ id: app().JobID }}
								class="text-sm font-medium text-primary hover:underline"
							>
								View job
							</Link>
						</div>
					</PageHeading>
					<JobNotesCard jobId={app().JobID} />
				</div>
			)}
		</QueryBoundary>
	);
}
