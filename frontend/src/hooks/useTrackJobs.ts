import { createMemo, createSignal } from "solid-js";
import { createJobColumns } from "@/components/jobs/columns";
import {
	toApplicationSummary,
	toExistingApp,
} from "@/components/jobs/TrackApplicationDialog";
import type { JobApplicationSummary } from "@/types/application";
import type { Job } from "@/types/job";
import { useApplications } from "./useApplications";
import { useDismissJob } from "./useDismissJob";

export function useTrackJobs(jobs: () => Job[]) {
	const applications = useApplications();
	const appsForJobs = createMemo<Record<string, JobApplicationSummary>>(() =>
		Object.fromEntries(
			(applications.data ?? []).map((app) => [
				app.JobID,
				toApplicationSummary(app),
			]),
		),
	);

	const [modalOpen, setModalOpen] = createSignal(false);
	const [trackingJobId, setTrackingJobId] = createSignal<string | null>(null);

	const openTrack = (jobId: string) => {
		setTrackingJobId(jobId);
		setModalOpen(true);
	};

	const currentJob = () => jobs().find((j) => j.ID === trackingJobId());
	const existingApp = () => {
		const app = applications.data?.find((a) => a.JobID === trackingJobId());
		return app ? toExistingApp(app) : undefined;
	};

	const dismiss = useDismissJob();
	const columns = createJobColumns({
		appsForJobs,
		onTrack: openTrack,
		onDismiss: dismiss.dismiss,
	});

	return {
		columns,
		applications,
		appsForJobs,
		openTrack,
		modalOpen,
		setModalOpen,
		currentJob,
		existingApp,
	};
}
