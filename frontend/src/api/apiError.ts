export class ApiError extends Error {
	constructor(
		readonly status: number,
		path: string,
		readonly body: unknown,
	) {
		const message = (body as { error?: unknown } | undefined)?.error;
		super(
			typeof message === "string" && message ? message : `${path}: ${status}`,
		);
	}
}
