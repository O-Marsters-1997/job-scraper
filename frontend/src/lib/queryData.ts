export function loadedData<T>(query: {
	isSuccess: boolean;
	data: T | undefined;
}): T | undefined {
	return query.isSuccess ? query.data : undefined;
}
