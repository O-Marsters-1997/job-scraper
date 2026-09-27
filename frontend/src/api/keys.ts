const jobsAll = ["jobs"] as const;
const scoresAll = ["scores"] as const;
const applicationsAll = ["applications"] as const;
const companiesAll = ["companies"] as const;

export const keys = {
	jobs: {
		all: jobsAll,
		page: () => [...jobsAll, "page"] as const,
		full: () => [...jobsAll, "full"] as const,
		detail: (id: string) => [...jobsAll, "detail", id] as const,
	},
	scores: {
		all: scoresAll,
		status: () => [...scoresAll, "status"] as const,
	},
	applications: {
		all: applicationsAll,
		byStatus: (statusId: string | undefined) =>
			[...applicationsAll, statusId ?? "all"] as const,
	},
	statuses: ["application-statuses"] as const,
	companies: {
		all: companiesAll,
		boards: (id: string) => [...companiesAll, "boards", id] as const,
	},
	sourceTargets: ["source-targets"] as const,
	sources: ["sources"] as const,
	profile: ["profile"] as const,
	aiPrefs: ["ai-prefs"] as const,
	cvTemplates: ["cv-templates"] as const,
	google: ["google-status"] as const,
	scoringConfig: ["scoring-config"] as const,
	scoringOptions: ["scoring-options"] as const,
	me: ["me"] as const,
} as const;
