import type { ScoringStatus } from "../types/scores";
import type { ScoringConfig } from "../types/scoringConfig";
import type {
	DimensionSpec,
	ScoringOption,
	ScoringOptionsView,
} from "../types/scoringOptions";

let scoringConfig: ScoringConfig = {
	notifyThreshold: 70,
	excludedTitleKeywords: [],
	excludedCompanies: [],
	excludedLocations: [],
	requiredLocations: [],
	requiredTitleKeywords: [],
	preferences: {
		picks: [
			{
				optionId: "tech:go",
				stance: "nice",
				source: "manual",
				overridden: false,
			},
			{
				optionId: "role:backend",
				stance: "nice",
				source: "manual",
				overridden: false,
			},
			{
				optionId: "domain:fintech",
				stance: "avoid",
				source: "manual",
				overridden: false,
			},
		],
		salaryFloor: null,
		preferenceText: "",
	},
	updatedAt: new Date("2024-01-01").toISOString(),
	backfillQueued: 0,
};

const scoringDimensions: DimensionSpec[] = [
	{ key: "tech", kind: "pair", stances: ["nice", "avoid"] },
	{ key: "role", kind: "pair", stances: ["nice", "avoid"] },
	{ key: "domain", kind: "pair", stances: ["nice", "avoid", "block"] },
	{ key: "seniority", kind: "multi", stances: ["nice"] },
	{ key: "work", kind: "multi", stances: ["nice"] },
	{ key: "stage", kind: "multi", stances: ["nice"] },
	{ key: "size", kind: "multi", stances: ["nice"] },
	{ key: "employment", kind: "multi", stances: ["nice"] },
];

const scoringOptions: ScoringOption[] = [
	{ id: "tech:go", dimension: "tech", label: "Go" },
	{ id: "tech:python", dimension: "tech", label: "Python" },
	{ id: "tech:typescript", dimension: "tech", label: "TypeScript" },
	{ id: "tech:rust", dimension: "tech", label: "Rust" },
	{ id: "tech:react", dimension: "tech", label: "React" },
	{ id: "tech:solidjs", dimension: "tech", label: "SolidJS" },
	{ id: "tech:postgres", dimension: "tech", label: "Postgres" },
	{ id: "tech:kafka", dimension: "tech", label: "Kafka" },
	{ id: "tech:kubernetes", dimension: "tech", label: "Kubernetes" },
	{ id: "tech:aws", dimension: "tech", label: "AWS" },
	{ id: "tech:terraform", dimension: "tech", label: "Terraform" },
	{ id: "tech:graphql", dimension: "tech", label: "GraphQL" },
	{ id: "role:backend", dimension: "role", label: "Backend" },
	{ id: "role:platform", dimension: "role", label: "Platform" },
	{ id: "role:full-stack", dimension: "role", label: "Full-stack" },
	{ id: "role:frontend", dimension: "role", label: "Frontend" },
	{ id: "role:sre", dimension: "role", label: "SRE" },
	{ id: "role:data-engineering", dimension: "role", label: "Data engineering" },
	{ id: "domain:fintech", dimension: "domain", label: "fintech" },
	{ id: "domain:devtools", dimension: "domain", label: "devtools" },
	{ id: "domain:health", dimension: "domain", label: "health" },
	{ id: "domain:climate", dimension: "domain", label: "climate" },
	{ id: "domain:crypto", dimension: "domain", label: "crypto" },
	{ id: "domain:gambling", dimension: "domain", label: "gambling" },
	{ id: "seniority:mid", dimension: "seniority", label: "Mid" },
	{ id: "seniority:senior", dimension: "seniority", label: "Senior" },
	{ id: "seniority:staff", dimension: "seniority", label: "Staff" },
	{ id: "work:remote", dimension: "work", label: "Remote" },
	{ id: "work:hybrid", dimension: "work", label: "Hybrid" },
	{ id: "work:onsite", dimension: "work", label: "Onsite" },
	{ id: "stage:seed", dimension: "stage", label: "Seed" },
	{ id: "stage:series-a", dimension: "stage", label: "Series A" },
	{ id: "stage:series-b", dimension: "stage", label: "Series B" },
	{ id: "stage:public", dimension: "stage", label: "Public" },
	{ id: "size:startup", dimension: "size", label: "Startup" },
	{ id: "size:scaleup", dimension: "size", label: "Scale-up" },
	{ id: "size:large", dimension: "size", label: "Large" },
	{ id: "size:enterprise", dimension: "size", label: "Enterprise" },
	{ id: "employment:permanent", dimension: "employment", label: "Permanent" },
	{ id: "employment:contract", dimension: "employment", label: "Contract" },
	{ id: "employment:part_time", dimension: "employment", label: "Part-time" },
];

export function getScoringConfig(): ScoringConfig {
	return structuredClone(scoringConfig);
}

export function getScoringOptions(): ScoringOptionsView {
	return { dimensions: scoringDimensions, options: scoringOptions };
}

export function getScoringStatus(): ScoringStatus {
	return { pending: 0 };
}

export function updateScoringConfig(payload: ScoringConfig): ScoringConfig {
	scoringConfig = payload;
	return scoringConfig;
}
