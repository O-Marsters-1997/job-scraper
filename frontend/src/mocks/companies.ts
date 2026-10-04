import { faker } from "@faker-js/faker";
import type { CompanyPageParams } from "@/api/companies";
import type {
	Company,
	CompanyBoard,
	CompanyExclusion,
	CompanyTracking,
	NewCompany,
	ReviewState,
	TrackedCompany,
} from "@/types/company";
import type {
	SourceTarget,
	UpdateSourceTargetPayload,
} from "@/types/sourceTarget";
import { isCompanyFavourite, setCompanyFavouriteFlag } from "./favourites";
import { failIfRequested, hostMatches, humanizeSlug, slugify } from "./helpers";
import { getJobs } from "./jobs";
import { excludeCompanyName, unexcludeCompanyName } from "./scoring";
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
		pinpoint: {
			host: "pinpointhq.com",
			url: (t) => `https://${t}.pinpointhq.com`,
		},
		teamtailor: {
			host: "teamtailor.com",
			url: (t) => `https://${t}.teamtailor.com`,
		},
		hibob: {
			host: "careers.hibob.com",
			url: (t) => `https://${t}.careers.hibob.com`,
		},
	};

const atsSourceForHost = (hostname: string) =>
	Object.entries(ATS_BOARDS).find(([, { host }]) =>
		hostMatches(hostname, host),
	)?.[0];

const withFavourite = (c: Company): Company => ({
	...c,
	Favourite: isCompanyFavourite(c.ID),
});

export function getCompanies(): Company[] {
	failIfRequested("getCompanies");
	return companies.map(withFavourite);
}

export function getCompanyPage(params: CompanyPageParams): {
	items: Company[];
	total: number;
} {
	failIfRequested("getCompanies");
	const q = (params.q ?? "").toLowerCase();
	const boardedCompanyIds = new Set(companyBoards.map((b) => b.CompanyID));
	const byName = (a: Company, b: Company) =>
		a.Name.localeCompare(b.Name) || a.ID.localeCompare(b.ID);
	const rank = (first: boolean) => (first ? 0 : 1);
	const byRelevance = (a: Company, b: Company) =>
		rank(a.Tracked) - rank(b.Tracked) ||
		rank(Boolean(a.Favourite) && a.Tracked) -
			rank(Boolean(b.Favourite) && b.Tracked) ||
		byName(a, b);
	const matches = getCompanies()
		.filter((c) => !params.tracked || c.Tracked)
		.filter((c) => !params.favourite || c.Favourite)
		.filter((c) => !params.noBoard || !boardedCompanyIds.has(c.ID))
		.filter(
			(c) =>
				c.Name.toLowerCase().includes(q) || c.Slug.toLowerCase().includes(q),
		)
		.sort(params.sort === "alphabetical" ? byName : byRelevance);
	const start = params.offset ?? 0;
	return {
		items: matches.slice(start, start + (params.limit ?? 50)),
		total: matches.length,
	};
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
				rollup: [
					{ dimension: "tech", label: "Go", yes: 2, known: 3, total: 4 },
					{ dimension: "domain", label: "Fintech", yes: 0, known: 0, total: 4 },
				],
				profile: null,
			};
		});
}

export function untrackCompany(id: string): void {
	if (!trackedCompanyIds.delete(id)) throw new Error("Company not tracked");
	companies = companies.map((c) =>
		c.ID === id ? { ...c, Tracked: false, TargetID: "" } : c,
	);
}

export function setCompanyFavourite(id: string, favourite: boolean): Company {
	const company = companies.find((c) => c.ID === id);
	if (!company) throw new Error("Company not found");
	setCompanyFavouriteFlag(id, favourite);
	const updated: Company =
		favourite && company.ReviewState === "new"
			? { ...company, ReviewState: "kept" }
			: company;
	companies = companies.map((c) => (c.ID === id ? updated : c));
	return withFavourite(updated);
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
	if (state === "dismissed") setCompanyFavouriteFlag(id, false);
	return {
		CompanyID: id,
		UserID: mockUser.id,
		Enabled: updated.Tracked,
		CheckIntervalMinutes: updated.CheckIntervalMinutes,
	};
}

export function excludeCompany(id: string): CompanyExclusion {
	const company = companies.find((c) => c.ID === id);
	if (!company) throw new Error("Company not found");
	const added = excludeCompanyName(company.Name);
	if (trackedCompanyIds.has(id)) setCompanyReview(id, "dismissed");
	return { name: company.Name, added };
}

export function unexcludeCompany(
	id: string,
	removeName: boolean,
): CompanyExclusion {
	const company = companies.find((c) => c.ID === id);
	if (!company) throw new Error("Company not found");
	if (removeName) unexcludeCompanyName(company.Name);
	if (trackedCompanyIds.has(id)) setCompanyReview(id, "new");
	return { name: company.Name, added: false };
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
		DisabledReason: "",
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
