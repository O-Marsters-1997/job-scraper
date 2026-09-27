import { keys } from "./keys";

function ok(cond: boolean, msg?: string): void {
	if (!cond) throw new Error(msg ?? "assertion failed");
}

function startsWith(
	nested: readonly unknown[],
	root: readonly unknown[],
): boolean {
	return root.every((value, i) => nested[i] === value);
}

ok(
	startsWith(keys.jobs.page(), keys.jobs.all),
	"jobs.page() must start with jobs.all",
);
ok(
	startsWith(keys.jobs.full(), keys.jobs.all),
	"jobs.full() must start with jobs.all",
);
ok(
	startsWith(keys.jobs.detail("job-1"), keys.jobs.all),
	"jobs.detail(id) must start with jobs.all",
);
ok(
	startsWith(keys.scores.status(), keys.scores.all),
	"scores.status() must start with scores.all",
);
ok(
	startsWith(keys.applications.byStatus("status-1"), keys.applications.all),
	"applications.byStatus(id) must start with applications.all",
);
ok(
	startsWith(keys.applications.byStatus(undefined), keys.applications.all),
	"applications.byStatus(undefined) must start with applications.all",
);
ok(
	startsWith(keys.companies.boards("company-1"), keys.companies.all),
	"companies.boards(id) must start with companies.all",
);

console.log("✓ keys checks passed");
