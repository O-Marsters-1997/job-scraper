import type {
	AddCompanyPayload,
	Company,
	CompanyBoard,
	CompanyTracking,
} from "../types/company";
import {
	companyBoardSchema,
	companySchema,
	companyTrackingSchema,
} from "../types/company";
import { apiFetch } from "./client";
import { API_BASE, mockDelay, mocked } from "./config";

export class UnresolvableBoardError extends Error {
	constructor() {
		super("could not resolve an ATS board from that URL");
	}
}

export async function fetchCompanies(): Promise<Company[]> {
	return mocked(
		async (db) => {
			await mockDelay();
			return db.getCompanies();
		},
		() => apiFetch("/companies", undefined, companySchema.array()),
	);
}

export async function addCompany(payload: AddCompanyPayload): Promise<Company> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			const company = db.addCompany(payload.url, payload.track ?? true);
			if (!company) throw new UnresolvableBoardError();
			return company;
		},
		async () => {
			const response = await fetch(`${API_BASE}/companies`, {
				method: "POST",
				credentials: "include",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify(payload),
			});
			if (response.status === 422) throw new UnresolvableBoardError();
			if (!response.ok)
				throw new Error(`Failed to add company: ${response.status}`);
			return companySchema.parse(await response.json());
		},
	);
}

export async function setCompanyTracking(
	id: string,
	enabled: boolean,
	checkIntervalMinutes?: number,
): Promise<CompanyTracking> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			return db.setCompanyTracking(id, enabled, checkIntervalMinutes);
		},
		() =>
			apiFetch(
				`/companies/${id}/tracking`,
				{
					method: "PUT",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({
						enabled,
						check_interval_minutes: checkIntervalMinutes,
					}),
				},
				companyTrackingSchema,
			),
	);
}

export async function fetchCompanyBoards(id: string): Promise<CompanyBoard[]> {
	return mocked(
		async (db) => {
			await mockDelay();
			return db.getCompanyBoards(id);
		},
		() =>
			apiFetch(
				`/companies/${id}/boards`,
				undefined,
				companyBoardSchema.array(),
			),
	);
}

export async function addCompanyBoard(
	id: string,
	url: string,
	confirm: boolean,
): Promise<CompanyBoard> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			const board = db.addCompanyBoard(id, url, confirm);
			if (!board) throw new UnresolvableBoardError();
			return board;
		},
		async () => {
			const response = await fetch(`${API_BASE}/companies/${id}/boards`, {
				method: "POST",
				credentials: "include",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ url, confirm }),
			});
			if (response.status === 422) throw new UnresolvableBoardError();
			if (response.status === 409)
				throw new Error("This board belongs to another company.");
			if (!response.ok)
				throw new Error(`Failed to add board: ${response.status}`);
			return companyBoardSchema.parse(await response.json());
		},
	);
}
