import type { Company, CompanyBoard, CompanyTracking } from "../types/company";
import { apiFetch } from "./client";
import { API_BASE, mockDelay, useMocks } from "./config";

export interface AddCompanyPayload {
	url: string;
	track?: boolean;
}

export class UnresolvableBoardError extends Error {
	constructor() {
		super("could not resolve an ATS board from that URL");
	}
}

export async function fetchCompanies(): Promise<Company[]> {
	if (useMocks()) {
		const { getCompanies } = await import("../mocks/db");
		await mockDelay();
		return getCompanies();
	}
	return apiFetch<Company[]>("/companies");
}

export async function addCompany(payload: AddCompanyPayload): Promise<Company> {
	if (useMocks()) {
		const { addCompany: mockAdd } = await import("../mocks/db");
		await mockDelay(80);
		const company = mockAdd(payload.url, payload.track ?? true);
		if (!company) throw new UnresolvableBoardError();
		return company;
	}
	const res = await fetch(`${API_BASE}/companies`, {
		method: "POST",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	});
	if (res.status === 422) throw new UnresolvableBoardError();
	if (!res.ok) throw new Error(`Failed to add company: ${res.status}`);
	return res.json();
}

export async function setCompanyTracking(
	id: string,
	enabled: boolean,
	checkIntervalMinutes?: number,
): Promise<CompanyTracking> {
	if (useMocks()) {
		const { setCompanyTracking: mockSetTracking } = await import("../mocks/db");
		await mockDelay(80);
		return mockSetTracking(id, enabled, checkIntervalMinutes);
	}
	return apiFetch<CompanyTracking>(`/companies/${id}/tracking`, {
		method: "PUT",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({
			enabled,
			check_interval_minutes: checkIntervalMinutes,
		}),
	});
}

export async function fetchCompanyBoards(id: string): Promise<CompanyBoard[]> {
	if (useMocks()) {
		const { getCompanyBoards } = await import("../mocks/db");
		await mockDelay();
		return getCompanyBoards(id);
	}
	return apiFetch<CompanyBoard[]>(`/companies/${id}/boards`);
}

export async function addCompanyBoard(
	id: string,
	url: string,
	confirm: boolean,
): Promise<CompanyBoard> {
	if (useMocks()) {
		const { addCompanyBoard: mockAdd } = await import("../mocks/db");
		await mockDelay(80);
		const board = mockAdd(id, url, confirm);
		if (!board) throw new UnresolvableBoardError();
		return board;
	}
	const res = await fetch(`${API_BASE}/companies/${id}/boards`, {
		method: "POST",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ url, confirm }),
	});
	if (res.status === 422) throw new UnresolvableBoardError();
	if (res.status === 409)
		throw new Error("This board belongs to another company.");
	if (!res.ok) throw new Error(`Failed to add board: ${res.status}`);
	return res.json();
}
