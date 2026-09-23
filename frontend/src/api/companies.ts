import type { Company, CompanyTracking } from "../types/company";
import { apiFetch } from "./client";
import { API_BASE, mockDelay, useMocks } from "./config";

export interface AddCompanyPayload {
	url: string;
	track?: boolean;
	scrape_now?: boolean;
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
