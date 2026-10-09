import type { IconName } from "@/components/Icon";

export interface SettingsSection {
	to: string;
	label: string;
	description: string;
	icon: IconName;
	children?: { to: string; label: string }[];
}

interface SettingsGroup {
	heading: string;
	adminOnly?: boolean;
	sections: SettingsSection[];
}

export const SETTINGS_SECTIONS: SettingsGroup[] = [
	{
		heading: "Account",
		sections: [
			{
				to: "/settings/profile",
				label: "Profile",
				description: "Username and email",
				icon: "user",
			},
			{
				to: "/settings/scoring",
				label: "Scoring",
				description: "What makes a job a good fit",
				icon: "pen",
				children: [
					{ to: "/settings/scoring/role", label: "Role" },
					{ to: "/settings/scoring/stack", label: "Tech & industry" },
					{ to: "/settings/scoring/filters", label: "Filters & alerts" },
					{ to: "/settings/scoring/other", label: "Other details" },
				],
			},
			{
				to: "/settings/ai",
				label: "AI",
				description: "OpenRouter API key",
				icon: "bot",
			},
		],
	},
	{
		heading: "Tracking",
		sections: [
			{
				to: "/settings/statuses",
				label: "Statuses",
				description: "Stages in your pipeline",
				icon: "settings",
			},
			{
				to: "/settings/searches",
				label: "Searches",
				description: "Boards and searches to scrape",
				icon: "search",
			},
		],
	},
	{
		heading: "Connections",
		sections: [
			{
				to: "/settings/integrations",
				label: "Integrations",
				description: "Google and other accounts",
				icon: "link",
			},
		],
	},
	{
		heading: "Admin",
		adminOnly: true,
		sections: [
			{
				to: "/settings/usage",
				label: "Usage",
				description: "Proxy quota",
				icon: "barChart",
			},
		],
	},
];

const ALL_SETTINGS_SECTIONS: SettingsSection[] = SETTINGS_SECTIONS.flatMap(
	(group) => group.sections,
);

export const SETTINGS_PAGES: string[] = ALL_SETTINGS_SECTIONS.flatMap(
	(section) => section.children?.map((child) => child.to) ?? [section.to],
);

export function findSettingsSection(
	pathname: string,
): SettingsSection | undefined {
	return ALL_SETTINGS_SECTIONS.find(
		(section) =>
			pathname === section.to || pathname.startsWith(`${section.to}/`),
	);
}
