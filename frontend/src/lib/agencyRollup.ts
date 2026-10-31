import type { CompanyKind } from "@/types/company";

export function isAgencyKind(kind: CompanyKind | undefined): boolean {
	return kind === "agency" || kind === "likely_agency";
}
