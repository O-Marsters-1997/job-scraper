const favouriteCompanyIds = new Set<string>();

export function isCompanyFavourite(companyId: string | undefined): boolean {
	return companyId !== undefined && favouriteCompanyIds.has(companyId);
}

export function setCompanyFavouriteFlag(
	companyId: string,
	favourite: boolean,
): void {
	if (favourite) favouriteCompanyIds.add(companyId);
	else favouriteCompanyIds.delete(companyId);
}
