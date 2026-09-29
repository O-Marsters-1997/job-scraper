import { faker } from "@faker-js/faker";
import type {
	Company,
	CompanyBoard,
	CompanyTracking,
	TrackedCompany,
} from "@/types/company";
import type {
	SourceTarget,
	UpdateSourceTargetPayload,
} from "@/types/sourceTarget";
import { failIfRequested, hostMatches, humanizeSlug, slugify } from "./helpers";
import { getJobs } from "./jobs";
import { seed } from "./seed";
import { mockUser } from "./user";

let companies: Company[] = seed.companies;
let companyBoards: CompanyBoard[] = seed.companyBoards;

const trackedCompanyIds = new Set(
	companies.filter((c) => c.Tracked).map((c) => c.ID),
);

const BOARD_URLS: Record<string, (token: string) => string> = {
	greenhouse: (t) => `https://boards.greenhouse.io/${t}`,
	lever: (t) => `https://jobs.lever.co/${t}`,
	ashby: (t) => `https://jobs.ashbyhq.com/${t}`,
	workable: (t) => `https://apply.workable.com/${t}`,
	recruitee: (t) => `https://${t}.recruitee.com`,
	personio: (t) => `https://${t}.jobs.personio.de`,
};

const ATS_HOSTS: Record<string, string> = {
	"greenhouse.io": "greenhouse",
	"lever.co": "lever",
	"ashbyhq.com": "ashby",
	"workable.com": "workable",
	"recruitee.com": "recruitee",
	"personio.de": "personio",
};

export function getCompanies(): Company[] {
	failIfRequested("getCompanies");
	return companies;
}

export function addCompany(url: string, track: boolean): Company | null {
	failIfRequested("addCompany");
	let hostname: string;
	try {
		hostname = new URL(url).hostname;
	} catch {
		return null;
	}
	const atsSource = Object.entries(ATS_HOSTS).find(([host]) =>
		hostMatches(hostname, host),
	)?.[1];
	if (!atsSource) return null;

	const token = url.replace(/\/$/, "").split("/").pop() ?? hostname;
	const slug = slugify(token);
	const company: Company = {
		ID: faker.string.uuid(),
		Slug: slug,
		Name: humanizeSlug(token),
		ATSSource: "",
		ATSToken: "",
		FirstSeenAt: new Date().toISOString(),
		JobCount: 0,
		Tracked: track,
		TargetID: track ? faker.string.uuid() : "",
		CheckIntervalMinutes: 360,
		LastCheckedAt: null,
	};
	companies = [company, ...companies];
	if (track) trackedCompanyIds.add(company.ID);
	addCompanyBoard(company.ID, url, false);
	return company;
}

export function getTrackedCompanies(): TrackedCompany[] {
	failIfRequested("getTrackedCompanies");
	return companies
		.filter((c) => trackedCompanyIds.has(c.ID))
		.map((c) => ({
			id: c.ID,
			name: c.Name,
			slug: c.Slug,
			enabled: c.Tracked,
			check_interval_minutes: c.CheckIntervalMinutes,
			boards: companyBoards
				.filter((b) => b.CompanyID === c.ID)
				.map((b) => ({
					id: b.ID,
					source: b.Source,
					board_token: b.BoardToken,
					status: b.Status,
					url: BOARD_URLS[b.Source]?.(b.BoardToken) ?? "",
				})),
			open_jobs: c.JobCount,
			relevant_jobs: getJobs().filter(
				(j) => j.CompanyID === c.ID && j.SuitabilityScore != null,
			).length,
			last_checked_at: c.LastCheckedAt,
		}))
		.sort((a, b) => a.name.localeCompare(b.name));
}

export function untrackCompany(id: string): void {
	if (!trackedCompanyIds.delete(id)) throw new Error("Company not tracked");
	companies = companies.map((c) =>
		c.ID === id ? { ...c, Tracked: false, TargetID: "" } : c,
	);
}

export function getCompanyBoards(companyID: string): CompanyBoard[] {
	return companyBoards.filter((board) => board.CompanyID === companyID);
}

export function addCompanyBoard(
	companyID: string,
	url: string,
	confirm: boolean,
): CompanyBoard | null {
	let hostname: string;
	try {
		hostname = new URL(url).hostname;
	} catch {
		return null;
	}
	const source = Object.entries(ATS_HOSTS).find(([host]) =>
		hostMatches(hostname, host),
	)?.[1];
	if (!source) return null;
	const token = url.replace(/\/$/, "").split("/").pop() ?? hostname;
	const existing = companyBoards.find(
		(board) => board.Source === source && board.BoardToken === token,
	);
	if (existing && existing.CompanyID !== companyID)
		throw new Error("This board belongs to another company.");
	const board: CompanyBoard = existing ?? {
		ID: faker.string.uuid(),
		CompanyID: companyID,
		Source: source,
		BoardToken: token,
		Status: "candidate",
		VerificationMethod: "",
		VerifiedAt: null,
		LastLinkedAt: null,
		RetiredAt: null,
		LastCompletedAt: null,
		CreatedAt: new Date().toISOString(),
	};
	if (confirm && board.Status === "candidate") {
		board.Status = "verified";
		board.VerificationMethod = "user_confirmed";
		board.VerifiedAt = new Date().toISOString();
	}
	if (!existing) companyBoards = [...companyBoards, board];
	return board;
}

export function setCompanyTracking(
	id: string,
	enabled: boolean,
	checkIntervalMinutes?: number,
): CompanyTracking {
	const idx = companies.findIndex((c) => c.ID === id);
	if (idx === -1) throw new Error("Company not found");
	const company = companies[idx]!;
	trackedCompanyIds.add(id);
	const updated: Company = {
		...company,
		Tracked: enabled,
		CheckIntervalMinutes:
			checkIntervalMinutes ?? (company.CheckIntervalMinutes || 360),
	};
	companies = [
		...companies.slice(0, idx),
		updated,
		...companies.slice(idx + 1),
	];
	return {
		CompanyID: id,
		UserID: mockUser.id,
		Enabled: enabled,
		CheckIntervalMinutes: updated.CheckIntervalMinutes,
	};
}

function companyToSourceTarget(company: Company): SourceTarget {
	return {
		ID: company.TargetID,
		UserID: "user-1",
		Source: company.ATSSource,
		Value: company.ATSToken,
		Enabled: company.Tracked,
		Filters: {},
		RunStatus: "idle",
		LastRunAt: company.LastCheckedAt,
		LastRunError: "",
		URL: "",
	};
}

export function updateCompanyTarget(
	id: string,
	patch: UpdateSourceTargetPayload,
): SourceTarget | null {
	const idx = companies.findIndex((c) => c.TargetID === id);
	if (idx === -1) return null;
	const company = companies[idx]!;
	const updated: Company = {
		...company,
		Tracked: patch.enabled ?? company.Tracked,
		CheckIntervalMinutes:
			patch.check_interval_minutes ?? company.CheckIntervalMinutes,
	};
	companies = [
		...companies.slice(0, idx),
		updated,
		...companies.slice(idx + 1),
	];
	return companyToSourceTarget(updated);
}
