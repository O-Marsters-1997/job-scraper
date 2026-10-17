export class ApiError extends Error {
	constructor(
		readonly status: number,
		path: string,
		readonly body: unknown,
	) {
		super(`${path}: ${status}`);
	}
}
