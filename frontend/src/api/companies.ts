import type {
	AddCompanyPayload,
	Company,
	CompanyBoard,
	CompanyExclusion,
	CompanyPage,
	CompanyTracking,
	NewCompany,
	ReviewState,
	TrackedCompany,
} from "../types/company";
import {
	companyBoardSchema,
	companyExclusionSchema,
	companyPageSchema,
	companySchema,
	companyTrackingSchema,
	newCompanySchema,
	trackedCompanySchema,
} from "../types/company";
import {
	ApiError,
	apiFetch,
	apiFetchVoid,
	jsonInit,
	rethrowStatus,
} from "./client";
import { mocked } from "./config";

export class UnresolvableBoardError extends Error {
	constructor() {
		super("could not resolve an ATS board from that URL");
	}
}

export type CompanyPageParams = {
	q?: string;
	tracked?: boolean;
	favourite?: boolean;
	cursor?: string;
	limit?: number;
};

export async function fetchCompanyPage(
	params: CompanyPageParams,
): Promise<CompanyPage> {
	return mocked(
		(db) => db.getCompanyPage(params),
		() => {
			const query = new URLSearchParams();
			if (params.q) query.set("q", params.q);
			if (params.tracked) query.set("tracked", "1");
			if (params.favourite) query.set("favourite", "1");
			if (params.cursor) query.set("cursor", params.cursor);
			if (params.limit) query.set("limit", String(params.limit));
			return apiFetch(`/companies?${query}`, companyPageSchema);
		},
	);
}

export async function fetchCompany(id: string): Promise<Company | null> {
	return mocked(
		(db) => db.getCompanies().find((c) => c.ID === id) ?? null,
		() =>
			apiFetch(`/companies/${encodeURIComponent(id)}`, companySchema).catch(
				(err) => {
					if (err instanceof ApiError && [400, 404].includes(err.status)) {
						return null;
					}
					throw err;
				},
			),
	);
}

export async function addCompany(payload: AddCompanyPayload): Promise<Company> {
	return mocked(
		(db) => {
			const company = db.addCompany(payload.url, payload.track ?? true);
			if (!company) throw new UnresolvableBoardError();
			return company;
		},
		() =>
			apiFetch("/companies", companySchema, jsonInit("POST", payload)).catch(
				rethrowStatus({ 422: () => new UnresolvableBoardError() }),
			),
	);
}

export async function setCompanyTracking(
	id: string,
	enabled: boolean,
	checkIntervalMinutes?: number,
): Promise<CompanyTracking> {
	return mocked(
		(db) => db.setCompanyTracking(id, enabled, checkIntervalMinutes),
		() =>
			apiFetch(
				`/companies/${id}/tracking`,
				companyTrackingSchema,
				jsonInit("PUT", {
					enabled,
					check_interval_minutes: checkIntervalMinutes,
				}),
			),
	);
}

export async function setCompanyFavourite(
	id: string,
	favourite: boolean,
): Promise<Company> {
	return mocked(
		(db) => db.setCompanyFavourite(id, favourite),
		() =>
			apiFetch(
				`/companies/${encodeURIComponent(id)}/favourite`,
				companySchema,
				{
					method: favourite ? "PUT" : "DELETE",
				},
			),
	);
}

export async function setCompanyReview(
	id: string,
	state: ReviewState,
): Promise<CompanyTracking> {
	return mocked(
		(db) => db.setCompanyReview(id, state),
		() =>
			apiFetch(
				`/companies/${id}/review`,
				companyTrackingSchema,
				jsonInit("PUT", { state }),
			),
	);
}

export async function excludeCompany(id: string): Promise<CompanyExclusion> {
	return mocked(
		(db) => db.excludeCompany(id),
		() =>
			apiFetch(
				`/companies/${id}/exclusion`,
				companyExclusionSchema,
				jsonInit("PUT", {}),
			),
	);
}

export async function unexcludeCompany(
	id: string,
	removeName: boolean,
): Promise<CompanyExclusion> {
	return mocked(
		(db) => db.unexcludeCompany(id, removeName),
		() =>
			apiFetch(
				`/companies/${id}/exclusion/undo`,
				companyExclusionSchema,
				jsonInit("POST", { removeName }),
			),
	);
}

export async function fetchCompanyBoards(id: string): Promise<CompanyBoard[]> {
	return mocked(
		(db) => db.getCompanyBoards(id),
		() => apiFetch(`/companies/${id}/boards`, companyBoardSchema.array()),
	);
}

export async function addCompanyBoard(
	id: string,
	url: string,
	confirm: boolean,
): Promise<CompanyBoard> {
	return mocked(
		(db) => {
			const board = db.addCompanyBoard(id, url, confirm);
			if (!board) throw new UnresolvableBoardError();
			return board;
		},
		() =>
			apiFetch(
				`/companies/${id}/boards`,
				companyBoardSchema,
				jsonInit("POST", { url, confirm }),
			).catch(
				rethrowStatus({
					422: () => new UnresolvableBoardError(),
					409: () => new Error("This board belongs to another company."),
				}),
			),
	);
}

export async function fetchTrackedCompanies(): Promise<TrackedCompany[]> {
	return mocked(
		(db) => db.getTrackedCompanies(),
		() => apiFetch("/companies/tracked", trackedCompanySchema.array()),
	);
}

export async function fetchNewCompanies(): Promise<NewCompany[]> {
	return mocked(
		(db) => db.getNewCompanies(),
		() => apiFetch("/companies/new", newCompanySchema.array()),
	);
}

export async function untrackCompany(
	id: string,
	opts?: { keepalive?: boolean },
): Promise<void> {
	return mocked(
		(db) => db.untrackCompany(id),
		() =>
			apiFetchVoid(`/companies/${id}/tracking`, {
				method: "DELETE",
				keepalive: opts?.keepalive ?? false,
			}),
	);
}
