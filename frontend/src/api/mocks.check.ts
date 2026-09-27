import { setDemoData } from "../lib/demoData";
import * as aiCredentials from "./aiCredentials";
import * as aiPrefs from "./aiPrefs";
import * as applicationStatuses from "./applicationStatuses";
import * as applications from "./applications";
import * as companies from "./companies";
import * as cvTemplates from "./cvTemplates";
import * as google from "./google";
import * as jobs from "./jobs";
import * as profile from "./profile";
import * as scores from "./scores";
import * as scoringConfig from "./scoringConfig";
import * as scoringOptions from "./scoringOptions";
import * as sources from "./sources";
import * as sourceTargets from "./sourceTargets";

setDemoData(true);

const originalFetch = globalThis.fetch;
let fetchCalled = "";
globalThis.fetch = (async (input: RequestInfo | URL) => {
	fetchCalled = String(input);
	throw new Error(`unexpected real fetch: ${fetchCalled}`);
}) as typeof fetch;

async function run(label: string, fn: () => Promise<unknown>): Promise<void> {
	fetchCalled = "";
	try {
		await fn();
	} catch {
		if (fetchCalled) throw new Error(`${label} reached fetch (${fetchCalled})`);
	}
	if (fetchCalled) throw new Error(`${label} reached fetch (${fetchCalled})`);
}

try {
	await run("fetchAllJobs", () => jobs.fetchAllJobs());
	await run("fetchJobs", () => jobs.fetchJobs());
	await run("fetchJob", () => jobs.fetchJob("job-1"));

	await run("fetchApplicationStatuses", () =>
		applicationStatuses.fetchApplicationStatuses(),
	);
	await run("createApplicationStatus", () =>
		applicationStatuses.createApplicationStatus("Test", "#000000"),
	);
	await run("updateApplicationStatus", () =>
		applicationStatuses.updateApplicationStatus("status-1", "Test", "#000000"),
	);
	await run("deleteApplicationStatus", () =>
		applicationStatuses.deleteApplicationStatus("status-1"),
	);

	await run("fetchApplications", () => applications.fetchApplications());
	await run("createApplication", () =>
		applications.createApplication({ job_id: "job-1" }),
	);
	await run("updateApplication", () =>
		applications.updateApplication("app-1", {}),
	);
	await run("deleteApplication", () => applications.deleteApplication("app-1"));

	await run("fetchCompanies", () => companies.fetchCompanies());
	await run("addCompany", () =>
		companies.addCompany({ url: "https://boards.greenhouse.io/acme" }),
	);
	await run("setCompanyTracking", () =>
		companies.setCompanyTracking("company-1", true),
	);
	await run("fetchCompanyBoards", () =>
		companies.fetchCompanyBoards("company-1"),
	);
	await run("addCompanyBoard", () =>
		companies.addCompanyBoard(
			"company-1",
			"https://boards.greenhouse.io/acme",
			false,
		),
	);

	await run("fetchCVTemplates", () => cvTemplates.fetchCVTemplates());
	await run("addTrackedDoc", () =>
		cvTemplates.addTrackedDoc("https://docs.google.com/document/d/abc"),
	);
	await run("removeTrackedDoc", () => cvTemplates.removeTrackedDoc("doc-1"));
	await run("hideTab", () => cvTemplates.hideTab("doc-1", "t.0"));
	await run("showTab", () => cvTemplates.showTab("doc-1", "t.0"));
	await run("fetchCVPdf", () => cvTemplates.fetchCVPdf("doc-1", "t.0"));

	await run("fetchAiPrefs", () => aiPrefs.fetchAiPrefs());
	await run("updateAiCredentials", () =>
		aiCredentials.updateAiCredentials({
			provider: "openrouter",
			apiKey: "key",
		}),
	);

	await run("fetchGoogleStatus", () => google.fetchGoogleStatus());
	await run("disconnectGoogle", () => google.disconnectGoogle());

	await run("fetchProfile", () => profile.fetchProfile());
	await run("updateProfile", () =>
		profile.updateProfile({ email: "demo@example.com" }),
	);

	await run("fetchSources", () => sources.fetchSources());
	await run("resolveBoard", () =>
		sources.resolveBoard("https://boards.greenhouse.io/acme"),
	);

	await run("fetchSourceTargets", () => sourceTargets.fetchSourceTargets());
	await run("createSourceTarget", () =>
		sourceTargets.createSourceTarget({
			source: "wis",
			value: "mocks-check-value",
		}),
	);
	await run("updateSourceTarget", () =>
		sourceTargets.updateSourceTarget("target-wis-1", { enabled: true }),
	);
	await run("deleteSourceTarget", () =>
		sourceTargets.deleteSourceTarget("mocks-check-nonexistent"),
	);
	await run("rerunSourceTarget", () =>
		sourceTargets.rerunSourceTarget("target-wis-1"),
	);

	await run("fetchScoringStatus", () => scores.fetchScoringStatus());
	await run("recomputeScores", () => scores.recomputeScores());

	const config = await scoringConfig.fetchScoringConfig();
	await run("fetchScoringConfig", () => scoringConfig.fetchScoringConfig());
	await run("updateScoringConfig", () =>
		scoringConfig.updateScoringConfig(config),
	);

	await run("fetchScoringOptions", () => scoringOptions.fetchScoringOptions());
} finally {
	globalThis.fetch = originalFetch;
	setDemoData(false);
}

console.log("✓ mocks checks passed");
