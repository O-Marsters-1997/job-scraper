import type { RowSelectionState } from "@tanstack/solid-table";
import { createMemo, createSignal } from "solid-js";
import { createJobColumns } from "@/components/jobs/columns";
import {
	toApplicationSummary,
	toExistingApp,
} from "@/components/jobs/TrackApplicationDialog";
import type { JobApplicationSummary } from "@/types/application";
import type { Job } from "@/types/job";
import { useApplications } from "./useApplications";
import { useSetCompanyFavourite } from "./useCompanies";
import { useDismissJob, useExcludeJobCompany } from "./useDismissJob";
import { useMarkJobsSeen } from "./useJobs";
import { announceBulkSeen } from "./useSeenToast";

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

	const [selection, setSelection] = createSignal<RowSelectionState>({});
	const [gradeJobs, setGradeJobs] = createSignal<Job[]>([]);
	const [gradeOpen, setGradeOpen] = createSignal(false);
	const openGrade = (target: Job[]) => {
		setGradeJobs(target);
		setGradeOpen(true);
	};

	const markSeen = useMarkJobsSeen();
	const bulkSeen = async (target: Job[], seen: boolean) => {
		await markSeen.mutateAsync({ jobIds: target.map((j) => j.ID), seen });
		setSelection({});
	};
	const markAllSeen = async (unseen: Job[]) => {
		const jobIds = unseen.map((j) => j.ID);
		await markSeen.mutateAsync({ jobIds, seen: true });
		announceBulkSeen(jobIds);
	};

	const dismiss = useDismissJob();
	const setFavourite = useSetCompanyFavourite();
	const exclude = useExcludeJobCompany();
	const columns = createJobColumns({
		appsForJobs,
		onTrack: openTrack,
		onDismiss: dismiss.dismiss,
		onGrade: (job) => openGrade([job]),
		onToggleFavourite: (job) => {
			if (!job.CompanyID) return;
			setFavourite.mutate({
				id: job.CompanyID,
				favourite: !job.CompanyFavourite,
			});
		},
		onExcludeCompany: exclude.exclude,
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
		selection,
		setSelection,
		gradeJobs,
		gradeOpen,
		setGradeOpen,
		openGrade,
		bulkSeen,
		markAllSeen,
	};
}
