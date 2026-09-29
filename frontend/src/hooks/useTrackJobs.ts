import { createMemo, createSignal } from "solid-js";
import { createJobColumns } from "@/components/jobs/columns";
import {
	toApplicationSummary,
	toExistingApp,
} from "@/components/jobs/TrackApplicationDialog";
import type { JobApplicationSummary } from "@/types/application";
import type { Job } from "@/types/job";
import { useApplications } from "./useApplications";

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
	const openEdit = (jobId: string) => {
		if (!appsForJobs()[jobId]) return;
		openTrack(jobId);
	};

	const currentJob = () => jobs().find((j) => j.ID === trackingJobId());
	const existingApp = () => {
		const app = applications.data?.find((a) => a.JobID === trackingJobId());
		return app ? toExistingApp(app) : undefined;
	};

	const columns = createJobColumns({
		appsForJobs,
		onTrack: openTrack,
		onEdit: openEdit,
	});

	return { columns, modalOpen, setModalOpen, currentJob, existingApp };
}
