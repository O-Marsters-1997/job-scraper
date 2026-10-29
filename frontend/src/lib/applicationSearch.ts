import type { ApplicationWithDetails } from "@/types/application";

export function matchesApplication(
	app: Pick<
		ApplicationWithDetails,
		"JobTitle" | "JobCompanySlug" | "JobLocation"
	>,
	query: string,
): boolean {
	const needle = query.trim().toLowerCase();
	if (!needle) return true;
	return [app.JobTitle, app.JobCompanySlug, app.JobLocation].some((field) =>
		field.toLowerCase().includes(needle),
	);
}
