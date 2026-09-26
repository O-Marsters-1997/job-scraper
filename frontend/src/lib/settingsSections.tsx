import type { JSX } from "solid-js";

export interface SettingsSection {
	to: string;
	label: string;
	description: string;
	icon: () => JSX.Element;
}

export interface SettingsGroup {
	heading: string;
	sections: SettingsSection[];
}

function ProfileIcon() {
	return (
		<svg
			aria-hidden="true"
			width="15"
			height="15"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
		>
			<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
			<circle cx="12" cy="7" r="4" />
		</svg>
	);
}

function ScoringIcon() {
	return (
		<svg
			aria-hidden="true"
			width="15"
			height="15"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
		>
			<path d="M12 20h9" />
			<path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z" />
		</svg>
	);
}

function AiIcon() {
	return (
		<svg
			aria-hidden="true"
			width="15"
			height="15"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
		>
			<path d="M12 2a2 2 0 0 1 2 2c0 .74-.4 1.39-1 1.73V7h1a7 7 0 0 1 7 7h1a1 1 0 0 1 1 1v3a1 1 0 0 1-1 1h-1v1a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-1H2a1 1 0 0 1-1-1v-3a1 1 0 0 1 1-1h1a7 7 0 0 1 7-7h1V5.73A2 2 0 0 1 10 4a2 2 0 0 1 2-2z" />
			<circle cx="9" cy="14" r="1" />
			<circle cx="15" cy="14" r="1" />
		</svg>
	);
}

function StatusesIcon() {
	return (
		<svg
			aria-hidden="true"
			width="15"
			height="15"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
		>
			<circle cx="12" cy="12" r="3" />
			<path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
		</svg>
	);
}

function SearchesIcon() {
	return (
		<svg
			aria-hidden="true"
			width="15"
			height="15"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
		>
			<circle cx="11" cy="11" r="8" />
			<line x1="21" y1="21" x2="16.65" y2="16.65" />
		</svg>
	);
}

function IntegrationsIcon() {
	return (
		<svg
			aria-hidden="true"
			width="15"
			height="15"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
		>
			<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
			<path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
		</svg>
	);
}

export const SETTINGS_SECTIONS: SettingsGroup[] = [
	{
		heading: "Account",
		sections: [
			{
				to: "/settings/profile",
				label: "Profile",
				description: "Your job-seeking preferences and CV context",
				icon: ProfileIcon,
			},
			{
				to: "/settings/scoring",
				label: "Scoring",
				description: "Profile, criteria and scale for job suitability scores",
				icon: ScoringIcon,
			},
			{
				to: "/settings/ai",
				label: "AI",
				description: "OpenRouter API key used to score jobs",
				icon: AiIcon,
			},
		],
	},
	{
		heading: "Tracking",
		sections: [
			{
				to: "/settings/statuses",
				label: "Statuses",
				description: "Custom stages for your application pipeline",
				icon: StatusesIcon,
			},
			{
				to: "/settings/searches",
				label: "Searches",
				description: "Discovery searches run automatically or on demand",
				icon: SearchesIcon,
			},
		],
	},
	{
		heading: "Connections",
		sections: [
			{
				to: "/settings/integrations",
				label: "Integrations",
				description: "Connect external accounts like Google",
				icon: IntegrationsIcon,
			},
		],
	},
];

const ALL_SETTINGS_SECTIONS: SettingsSection[] = SETTINGS_SECTIONS.flatMap(
	(group) => group.sections,
);

export function findSettingsSection(
	pathname: string,
): SettingsSection | undefined {
	return ALL_SETTINGS_SECTIONS.find((section) => section.to === pathname);
}
