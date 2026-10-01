import { faker } from "@faker-js/faker";
import type {
	Company,
	CompanyBoard,
	CompanyTracking,
	NewCompany,
	ReviewState,
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

const ATS_BOARDS: Record<string, { host: string; url: (t: string) => string }> =
	{
		greenhouse: {
			host: "greenhouse.io",
			url: (t) => `https://boards.greenhouse.io/${t}`,
		},
		lever: { host: "lever.co", url: (t) => `https://jobs.lever.co/${t}` },
		ashby: { host: "ashbyhq.com", url: (t) => `https://jobs.ashbyhq.com/${t}` },
		workable: {
			host: "workable.com",
			url: (t) => `https://apply.workable.com/${t}`,
		},
		recruitee: {
			host: "recruitee.com",
			url: (t) => `https://${t}.recruitee.com`,
		},
		personio: {
			host: "personio.de",
			url: (t) => `https://${t}.jobs.personio.de`,
		},
	};

const atsSourceForHost = (hostname: string) =>
	Object.entries(ATS_BOARDS).find(([, { host }]) =>
		hostMatches(hostname, host),
	)?.[0];

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
	const atsSource = atsSourceForHost(hostname);
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
		ReviewState: track ? "kept" : "",
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
			review_state: c.ReviewState || "kept",
			check_interval_minutes: c.CheckIntervalMinutes,
			boards: companyBoards
				.filter((b) => b.CompanyID === c.ID)
				.map((b) => ({
					id: b.ID,
					source: b.Source,
					board_token: b.BoardToken,
					status: b.Status,
					url: ATS_BOARDS[b.Source]?.url(b.BoardToken) ?? "",
				})),
			open_jobs: c.JobCount,
			relevant_jobs: getJobs().filter(
				(j) => j.CompanyID === c.ID && j.SuitabilityScore != null,
			).length,
			last_checked_at: c.LastCheckedAt,
		}))
		.sort((a, b) => a.name.localeCompare(b.name));
}

export function getNewCompanies(): NewCompany[] {
	failIfRequested("getNewCompanies");
	return getTrackedCompanies()
		.filter((c) => c.review_state === "new")
		.map((c) => {
			const scores = getJobs()
				.filter((j) => j.CompanyID === c.id && j.SuitabilityScore != null)
				.map((j) => j.SuitabilityScore as number);
			return {
				id: c.id,
				name: c.name,
				slug: c.slug,
				boards: c.boards,
				matching_roles: c.open_jobs,
				best_suitability: scores.length ? Math.max(...scores) : null,
				profile: [
					{ dimension: "tech", label: "Go", yes: 2, known: 3, total: 4 },
					{ dimension: "domain", label: "Fintech", yes: 0, known: 0, total: 4 },
				],
			};
		});
}

export function untrackCompany(id: string): void {
	if (!trackedCompanyIds.delete(id)) throw new Error("Company not tracked");
	companies = companies.map((c) =>
		c.ID === id ? { ...c, Tracked: false, TargetID: "" } : c,
	);
}

export function setCompanyReview(
	id: string,
	state: ReviewState,
): CompanyTracking {
	const company = companies.find((c) => c.ID === id);
	if (!company || !trackedCompanyIds.has(id))
		throw new Error("Company not tracked");
	const updated: Company = {
		...company,
		Tracked: state !== "dismissed",
		ReviewState: state,
	};
	companies = companies.map((c) => (c.ID === id ? updated : c));
	return {
		CompanyID: id,
		UserID: mockUser.id,
		Enabled: updated.Tracked,
		CheckIntervalMinutes: updated.CheckIntervalMinutes,
	};
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
	const source = atsSourceForHost(hostname);
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
