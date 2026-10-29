const jobsAll = ["jobs"] as const;
const scoresAll = ["scores"] as const;
const applicationsAll = ["applications"] as const;
const companiesAll = ["companies"] as const;
const tailoringAll = ["tailoring"] as const;
const tasksAll = ["tasks"] as const;

export const keys = {
	jobs: {
		all: jobsAll,
		page: () => [...jobsAll, "page"] as const,
		list: (params: object) => [...jobsAll, "page", params] as const,
		insights: () => [...jobsAll, "insights"] as const,
		overview: (ranked: boolean) => [...jobsAll, "overview", ranked] as const,
		detail: (id: string) => [...jobsAll, "detail", id] as const,
	},
	scores: {
		all: scoresAll,
		status: () => [...scoresAll, "status"] as const,
	},
	applications: {
		all: applicationsAll,
		list: (statusId: string | undefined, chase: boolean) =>
			[...applicationsAll, statusId ?? "all", chase] as const,
	},
	tasks: {
		all: tasksAll,
		list: (params: object) => [...tasksAll, "list", params] as const,
		types: [...tasksAll, "types"] as const,
		count: [...tasksAll, "count"] as const,
		forJob: (jobId: string) => [...tasksAll, "job", jobId] as const,
	},
	statuses: ["application-statuses"] as const,
	jobNote: (jobId: string) => ["job-note", jobId] as const,
	jobComments: (jobId: string) => ["job-comments", jobId] as const,
	recruiterContacts: (jobId: string) => ["recruiter-contacts", jobId] as const,
	recruiterContactSummaries: {
		all: ["recruiter-contacts", "summaries"] as const,
		for: (jobIds: readonly string[]) =>
			["recruiter-contacts", "summaries", jobIds] as const,
	},
	companies: {
		all: companiesAll,
		list: (params: object) => [...companiesAll, "list", params] as const,
		search: (q: string) => [...companiesAll, "search", q] as const,
		detail: (id: string) => [...companiesAll, "detail", id] as const,
		atsCounts: [...companiesAll, "ats-counts"] as const,
		tracked: [...companiesAll, "tracked"] as const,
		new: [...companiesAll, "new"] as const,
		newPage: (page: number) => [...companiesAll, "new", page] as const,
		boards: (id: string) => [...companiesAll, "boards", id] as const,
	},
	experience: ["experience"] as const,
	bankSkills: ["bank-skills"] as const,
	letterSkeleton: ["letter-skeleton"] as const,
	coverLetters: (jobId: string) => ["cover-letters", jobId] as const,
	tailoring: {
		all: tailoringAll,
		headings: (docId: string, tabId: string) =>
			[...tailoringAll, "headings", docId, tabId] as const,
		suggestions: (jobId: string, docId: string, tabId: string) =>
			[...tailoringAll, "suggestions", jobId, docId, tabId] as const,
		skillSuggestions: (jobId: string, docId: string, tabId: string) =>
			[...tailoringAll, "skill-suggestions", jobId, docId, tabId] as const,
		experienceMatch: (jobId: string) =>
			[...tailoringAll, "experience-match", jobId] as const,
		draft: (id: string) => [...tailoringAll, "draft", id] as const,
		draftLayout: (id: string) => [...tailoringAll, "draft-layout", id] as const,
		draftLean: (id: string) => [...tailoringAll, "draft-lean", id] as const,
		draftRoles: (id: string) => [...tailoringAll, "draft-roles", id] as const,
		draftInsights: (id: string) =>
			[...tailoringAll, "draft-insights", id] as const,
		jobDrafts: (jobId: string) =>
			[...tailoringAll, "job-drafts", jobId] as const,
	},
	sourceTargets: ["source-targets"] as const,
	scrapeRuns: (params: object) => ["scrape-runs", params] as const,
	sourceHealth: ["scrape-runs", "health"] as const,
	scrapeRunJobSnapshots: (runId: string) =>
		["scrape-runs", "job-changes", runId] as const,
	scrapeRunSummary: (params: object) =>
		["scrape-runs", "summary", params] as const,
	sources: ["sources"] as const,
	profile: ["profile"] as const,
	aiPrefs: ["ai-prefs"] as const,
	aiUsage: ["ai-usage"] as const,
	proxyUsage: ["proxy-usage"] as const,
	cvTemplates: ["cv-templates"] as const,
	google: ["google-status"] as const,
	scoringConfig: ["scoring-config"] as const,
	scoringOptions: ["scoring-options"] as const,
	playgroundQueue: ["scoring-playground-queue"] as const,
	scoringPreview: (preferences: object) =>
		["scoring-preview", preferences] as const,
	grade: (jobId: string) => ["grade", jobId] as const,
	scoreFeedback: ["score-feedback"] as const,
	me: ["me"] as const,
} as const;
