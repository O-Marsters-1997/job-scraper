import type { ScoreRow } from "@/types/job";

export const COMPANIES = [
	"Monzo",
	"Figma",
	"Revolut",
	"Stripe",
	"Deliveroo",
	"Wise",
	"Canva",
	"Shopify",
	"Notion",
	"Cloudflare",
	"Linear",
	"Netflix",
	"Vercel",
	"Supabase",
	"GitHub",
	"GitLab",
	"Atlassian",
	"HubSpot",
	"Intercom",
	"Zendesk",
	"Snowflake",
	"Databricks",
	"MongoDB",
	"Elastic",
	"Datadog",
	"HashiCorp",
	"Airbnb",
	"Spotify",
	"Twilio",
	"Slack",
	"PlanetScale",
	"Retool",
	"Contentful",
	"Figma",
];

export const LOCATIONS = [
	"Remote",
	"Remote",
	"London, UK",
	"London, UK",
	"Dublin, IE",
	"Berlin, DE",
	"Amsterdam, NL",
	"New York, US",
	"San Francisco, US",
	"Toronto, CA",
];

export const SOURCES = [
	"LinkedIn",
	"LinkedIn",
	"LinkedIn",
	"LinkedIn",
	"Indeed",
	"Indeed",
	"Greenhouse",
	"Greenhouse",
	"Lever",
];

export const JOB_DESCRIPTIONS = [
	`We're building the infrastructure that powers payments for millions of people. As a key member of our engineering team, you'll design, build, and scale distributed systems that handle real-time financial transactions at global scale.\n\nYou'll work closely with product, data, and design teams to ship features end-to-end — from architecture decisions to production monitoring. We operate a 'you build it, you run it' culture, so you'll own your services in production.\n\nWe're looking for engineers who care deeply about reliability, security, and developer experience. You'll be joining a team of 8 engineers embedded in a product squad, shipping roughly every two weeks.`,
	`Our mission is to make design accessible to everyone. You'll join a small, high-ownership team building the tools that millions of designers and developers use every day.\n\nThis role is fully remote-first. We invest heavily in async communication and documentation. You'll be expected to write clearly, work autonomously, and raise the bar for your team through code review, RFCs, and mentorship.\n\nWe ship frequently, measure impact rigorously, and give engineers real ownership over the systems they build.`,
	`We're reimagining how companies manage their finances. You'll work on core product features — from the first login to reconciliation workflows — used by finance teams across thousands of businesses.\n\nEngineering here means owning problems, not tickets. We expect you to talk to customers, influence the roadmap, and contribute to a culture of technical excellence.\n\nYour team is responsible for the full stack: API design, database modelling, and the frontend surfaces your users interact with every day.`,
	`Join a team that's rethinking how the internet is built. We run one of the world's largest networks, and your code will run in data centres across 300+ cities.\n\nYou'll build tooling, services, and systems that our entire engineering organisation depends on. We value people who can operate at multiple levels of abstraction — from TCP to product experience.\n\nThis is not a maintenance role. We're actively rebuilding core infrastructure and expect strong opinions about how things should work.`,
	`We believe great software is built by people who understand the problem deeply. You'll join a product-focused engineering team shipping features that help millions of professionals find the right opportunities and build meaningful careers.\n\nWe move fast but deliberately — we write RFCs for significant changes, run blameless post-mortems, and keep our on-call burden low through good design. You'll have time to think, not just to ship.`,
];

export const SKILL_SETS = [
	["TypeScript", "React", "CSS", "Node.js", "PostgreSQL"],
	["Go", "PostgreSQL", "gRPC", "Docker", "Kubernetes"],
	["Python", "SQL", "Spark", "Airflow", "dbt"],
	["TypeScript", "Go", "PostgreSQL", "Docker", "CI/CD"],
	["Kotlin", "Java", "Spring Boot", "AWS", "Kafka"],
	["Swift", "SwiftUI", "iOS SDK", "Xcode", "Objective-C"],
	["Python", "PyTorch", "TensorFlow", "SQL", "MLflow"],
	["Terraform", "Kubernetes", "Prometheus", "AWS", "Grafana"],
];

export const EMPLOYMENT_TYPES = [
	"Full-time",
	"Full-time",
	"Full-time",
	"Contract",
];
export const TEAM_NAMES = [
	"Platform",
	"Growth",
	"Infrastructure",
	"Product Engineering",
	"Data & Analytics",
	"Security",
	"Developer Experience",
	"Mobile",
	"Core Services",
];
export const COMPANY_SIZES = [
	"50–200",
	"200–500",
	"500–2,000",
	"2,000–10,000",
	"10,000+",
];
export const SALARY_RANGES = [
	"£60,000–£80,000",
	"£80,000–£110,000",
	"£110,000–£150,000",
	"$130,000–$170,000",
	"£70,000–£90,000",
	"Competitive + equity",
];

export function deriveExperienceLevel(title: string): string {
	const t = title.toLowerCase();
	if (t.includes("principal") || t.includes("staff"))
		return "Principal / Staff";
	if (t.includes("vp") || t.includes("manager") || t.includes("lead"))
		return "Leadership";
	if (t.includes("senior")) return "Senior (5+ years)";
	return "Mid-level (2–5 years)";
}

export const JOB_TITLES = [
	"Software Engineer",
	"Senior Software Engineer",
	"Staff Engineer",
	"Principal Engineer",
	"Frontend Engineer",
	"Senior Frontend Engineer",
	"Full Stack Engineer",
	"Senior Full Stack Engineer",
	"Backend Engineer",
	"Senior Backend Engineer",
	"Platform Engineer",
	"DevOps Engineer",
	"Site Reliability Engineer",
	"Infrastructure Engineer",
	"Data Engineer",
	"Senior Data Engineer",
	"Machine Learning Engineer",
	"Product Designer",
	"Senior Product Designer",
	"UX Engineer",
	"Engineering Manager",
	"Senior Engineering Manager",
	"VP of Engineering",
	"Product Manager",
	"Senior Product Manager",
	"Technical Program Manager",
	"iOS Engineer",
	"Android Engineer",
	"Mobile Engineer",
	"Security Engineer",
	"Staff Security Engineer",
	"QA Engineer",
	"Solutions Architect",
	"Data Scientist",
	"Senior Data Scientist",
	"Analytics Engineer",
	"Business Intelligence Engineer",
	"Software Development Engineer in Test",
];

export const BREAKDOWN_PICKS: {
	key: string;
	label: string;
	stance: "nice" | "avoid";
}[] = [
	{ key: "tech:go", label: "Go", stance: "nice" },
	{ key: "tech:python", label: "Python", stance: "nice" },
	{ key: "role:backend", label: "Backend", stance: "nice" },
	{ key: "domain:fintech", label: "fintech", stance: "avoid" },
];

export function mockBreakdown(i: number): ScoreRow[] {
	return BREAKDOWN_PICKS.map((p, j) => {
		const roll = (i + j) % 3;
		if (roll === 0) {
			return {
				key: p.key,
				label: p.label,
				stance: p.stance,
				resolved: "unknown",
				effect: "unknown",
				overridden: false,
			} satisfies ScoreRow;
		}
		if (p.stance === "nice") {
			const matched = roll === 1;
			return {
				key: p.key,
				label: p.label,
				stance: p.stance,
				resolved: matched ? "yes" : "no",
				effect: matched ? "meets" : "misses",
				overridden: false,
			} satisfies ScoreRow;
		}
		const hit = roll === 1;
		return {
			key: p.key,
			label: p.label,
			stance: p.stance,
			resolved: hit ? "yes" : "no",
			effect: hit ? "misses" : "neutral",
			overridden: false,
		} satisfies ScoreRow;
	});
}

export const ATS_SOURCES = [
	"greenhouse",
	"lever",
	"ashby",
	"workable",
	"recruitee",
	"personio",
];

export const APP_DISTRIBUTION: Array<{ statusIndex: number; count: number }> = [
	{ statusIndex: 0, count: 2 },
	{ statusIndex: 1, count: 5 },
	{ statusIndex: 2, count: 2 },
	{ statusIndex: 3, count: 1 },
	{ statusIndex: 4, count: 1 },
	{ statusIndex: 5, count: 1 },
];
