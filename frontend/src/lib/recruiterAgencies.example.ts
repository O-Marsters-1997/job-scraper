import { fetchCompanyPage, setCompanyKind } from "@/api/companies";
import {
	fetchRecruiterContacts,
	fetchRecruiterContactsForJobs,
	logRecruiterContact,
} from "@/api/recruiterContacts";
import type { Job } from "@/types/job";
import type { RecruiterContact } from "@/types/recruiterContact";
import { type AgencyRollup, agencyRollup } from "./agencyRollup";
import { showsMaybeRecruiter } from "./companyKind";
import { applyJobFilters, DEFAULT_FILTERS } from "./jobFilters";

export async function j1MarkRecruiter(job: Job): Promise<void> {
	if (job.CompanyID) await setCompanyKind(job.CompanyID, "agency");
}

export async function j2UndoMark(job: Job): Promise<void> {
	if (job.CompanyID) await setCompanyKind(job.CompanyID, "direct");
}

export function j3DirectOnly(jobs: Job[]): Job[] {
	return applyJobFilters(jobs, { ...DEFAULT_FILTERS, poster: "direct" });
}

export async function j4AgencyRoles(jobs: Job[]): Promise<Job[]> {
	const agencies = await fetchCompanyPage({ agency: true });
	const agency = agencies.items[0];
	return agency ? jobs.filter((j) => j.CompanyID === agency.ID) : [];
}

export function j5MaybeRecruiters(jobs: Job[]): Job[] {
	return jobs.filter(showsMaybeRecruiter);
}

export async function j6LogContact(job: Job): Promise<RecruiterContact[]> {
	await logRecruiterContact({
		jobId: job.ID,
		contactedOn: "2026-10-12",
		channel: "linkedin_message",
		outcome: "no_reply",
	});
	return fetchRecruiterContacts(job.ID);
}

export async function j7AgencyRollup(agencyJobs: Job[]): Promise<AgencyRollup> {
	const summaries = await fetchRecruiterContactsForJobs(
		agencyJobs.map((j) => j.ID),
	);
	return agencyRollup(agencyJobs, summaries);
}
