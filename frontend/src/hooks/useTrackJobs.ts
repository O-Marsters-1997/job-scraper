import { createMemo, createSignal } from "solid-js";
import {
	createJobColumns,
	type JobTableContext,
} from "@/components/jobs/columns";
import {
	toApplicationSummary,
	toExistingApp,
} from "@/components/jobs/TrackApplicationDialog";
import { loadedData } from "@/lib/queryData";
import type { JobApplicationSummary } from "@/types/application";
import type { Job } from "@/types/job";
import type { RecruiterContactSummary } from "@/types/recruiterContact";
import { useApplications } from "./useApplications";
import {
	useSetCompaniesKind,
	useSetCompanyFavourite,
	useSetCompanyKind,
} from "./useCompanies";
import {
	announceNotice,
	companyIdsOf,
	describeKind,
	useDismissJob,
	useExcludeJobCompanies,
	useExcludeJobCompany,
} from "./useDismissJob";
import { useMarkJobsSeen } from "./useJobs";
import { announceBulkSeen } from "./useSeenToast";

export function useTrackJobs(
	jobs: () => Job[],
	contactSummaries: () =>
		| Record<string, RecruiterContactSummary>
		| undefined = () => undefined,
	clearSelection: () => void = () => undefined,
) {
	const applications = useApplications();
	const appsForJobs = createMemo<Record<string, JobApplicationSummary>>(() =>
		Object.fromEntries(
			(loadedData(applications) ?? []).map((app) => [
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
		const app = loadedData(applications)?.find(
			(a) => a.JobID === trackingJobId(),
		);
		return app ? toExistingApp(app) : undefined;
	};

	const [gradeJobs, setGradeJobs] = createSignal<Job[]>([]);
	const [gradeOpen, setGradeOpen] = createSignal(false);
	const openGrade = (target: Job[]) => {
		setGradeJobs(target);
		setGradeOpen(true);
	};

	const markSeen = useMarkJobsSeen();
	const bulkSeen = async (target: Job[], seen: boolean) => {
		await markSeen.mutateAsync({ jobIds: target.map((j) => j.ID), seen });
		clearSelection();
	};
	const markAllSeen = async (unseen: Job[]) => {
		const jobIds = unseen.map((j) => j.ID);
		await markSeen.mutateAsync({ jobIds, seen: true });
		announceBulkSeen(jobIds);
	};

	const bulkExclude = useExcludeJobCompanies();
	const setKinds = useSetCompaniesKind();
	const bulkSetKind = async (target: Job[], kind: "agency" | "direct") => {
		const { ids, skipped } = companyIdsOf(target);
		if (ids.length === 0) {
			announceNotice(describeKind(0, kind, skipped));
			return;
		}
		try {
			const result = await setKinds.mutateAsync({ ids, kind });
			announceNotice(describeKind(result.updated, kind, skipped));
		} catch {
			announceNotice("Could not update companies.");
		}
	};

	const dismiss = useDismissJob();
	const setFavourite = useSetCompanyFavourite();
	const exclude = useExcludeJobCompany();
	const setKind = useSetCompanyKind();
	const columnContext: JobTableContext = {
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
		onSetCompanyKind: (job, kind) => {
			if (!job.CompanyID) return;
			setKind.mutate({ id: job.CompanyID, kind });
		},
	};
	const columns = createJobColumns(columnContext);
	const contactColumns = createJobColumns({
		...columnContext,
		contactSummaries,
	});

	return {
		columns,
		contactColumns,
		applications,
		appsForJobs,
		openTrack,
		modalOpen,
		setModalOpen,
		currentJob,
		existingApp,
		gradeJobs,
		gradeOpen,
		setGradeOpen,
		openGrade,
		bulkSeen,
		markAllSeen,
		bulkExcludeCompanies: bulkExclude.exclude,
		bulkMarkRecruiters: (target: Job[]) => bulkSetKind(target, "agency"),
	};
}
