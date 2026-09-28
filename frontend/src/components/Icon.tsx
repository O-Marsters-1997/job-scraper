import type { JSX } from "solid-js";

const ICONS = {
	alert: () => (
		<>
			<path d="M12 9v4" />
			<path d="M12 17h.01" />
			<path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0Z" />
		</>
	),
	arrowRight: () => <path d="M5 12h14M13 6l6 6-6 6" />,
	barChart: () => (
		<>
			<line x1="18" y1="20" x2="18" y2="10" />
			<line x1="12" y1="20" x2="12" y2="4" />
			<line x1="6" y1="20" x2="6" y2="14" />
		</>
	),
	bot: () => (
		<>
			<path d="M12 2a2 2 0 0 1 2 2c0 .74-.4 1.39-1 1.73V7h1a7 7 0 0 1 7 7h1a1 1 0 0 1 1 1v3a1 1 0 0 1-1 1h-1v1a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-1H2a1 1 0 0 1-1-1v-3a1 1 0 0 1 1-1h1a7 7 0 0 1 7-7h1V5.73A2 2 0 0 1 10 4a2 2 0 0 1 2-2z" />
			<circle cx="9" cy="14" r="1" />
			<circle cx="15" cy="14" r="1" />
		</>
	),
	briefcase: () => (
		<>
			<rect x="2" y="7" width="20" height="14" rx="2" ry="2" />
			<path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16" />
		</>
	),
	building: () => (
		<>
			<path d="M3 21h18" />
			<path d="M5 21V7l7-4 7 4v14" />
			<path d="M9 9h1" />
			<path d="M14 9h1" />
			<path d="M9 13h1" />
			<path d="M14 13h1" />
			<path d="M9 21v-4h6v4" />
		</>
	),
	check: () => <polyline points="20 6 9 17 4 12" />,
	chevronDown: () => <polyline points="6 9 12 15 18 9" />,
	chevronLeft: () => <polyline points="15 18 9 12 15 6" />,
	chevronRight: () => <polyline points="9 18 15 12 9 6" />,
	dashboard: () => (
		<>
			<rect width="7" height="9" x="3" y="3" rx="1" />
			<rect width="7" height="5" x="14" y="3" rx="1" />
			<rect width="7" height="9" x="14" y="12" rx="1" />
			<rect width="7" height="5" x="3" y="16" rx="1" />
		</>
	),
	dollar: () => (
		<>
			<line x1="12" y1="1" x2="12" y2="23" />
			<path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
		</>
	),
	externalLink: () => (
		<>
			<path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
			<polyline points="15 3 21 3 21 9" />
			<line x1="10" y1="14" x2="21" y2="3" />
		</>
	),
	file: () => (
		<>
			<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
			<polyline points="14 2 14 8 20 8" />
		</>
	),
	fileLines: () => (
		<>
			<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
			<polyline points="14 2 14 8 20 8" />
			<line x1="16" y1="13" x2="8" y2="13" />
			<line x1="16" y1="17" x2="8" y2="17" />
		</>
	),
	fileText: () => (
		<>
			<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
			<polyline points="14 2 14 8 20 8" />
			<line x1="16" y1="13" x2="8" y2="13" />
			<line x1="16" y1="17" x2="8" y2="17" />
			<polyline points="10 9 9 9 8 9" />
		</>
	),
	filter: () => (
		<>
			<line x1="4" y1="6" x2="20" y2="6" />
			<line x1="8" y1="12" x2="16" y2="12" />
			<line x1="11" y1="18" x2="13" y2="18" />
		</>
	),
	history: () => (
		<>
			<path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
			<path d="M3 3v5h5" />
		</>
	),
	home: () => (
		<>
			<path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
			<polyline points="9 22 9 12 15 12 15 22" />
		</>
	),
	link: () => (
		<>
			<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
			<path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
		</>
	),
	logOut: () => (
		<>
			<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
			<polyline points="16 17 21 12 16 7" />
			<line x1="21" y1="12" x2="9" y2="12" />
		</>
	),
	menu: () => (
		<>
			<line x1="3" y1="6" x2="21" y2="6" />
			<line x1="3" y1="12" x2="21" y2="12" />
			<line x1="3" y1="18" x2="21" y2="18" />
		</>
	),
	moreVertical: () => (
		<>
			<circle cx="12" cy="5" r="1.5" fill="currentColor" stroke="none" />
			<circle cx="12" cy="12" r="1.5" fill="currentColor" stroke="none" />
			<circle cx="12" cy="19" r="1.5" fill="currentColor" stroke="none" />
		</>
	),
	pen: () => (
		<>
			<path d="M12 20h9" />
			<path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z" />
		</>
	),
	plus: () => <path d="M12 5v14M5 12h14" />,
	rotateCcw: () => (
		<>
			<path d="M1 4v6h6" />
			<path d="M3.51 15a9 9 0 1 0 .49-3.5" />
		</>
	),
	search: () => (
		<>
			<circle cx="11" cy="11" r="8" />
			<line x1="21" y1="21" x2="16.65" y2="16.65" />
		</>
	),
	settings: () => (
		<>
			<circle cx="12" cy="12" r="3" />
			<path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
		</>
	),
	sliders: () => (
		<>
			<line x1="21" y1="4" x2="7" y2="4" />
			<line x1="3" y1="4" x2="7" y2="4" />
			<line x1="21" y1="12" x2="11" y2="12" />
			<line x1="7" y1="12" x2="3" y2="12" />
			<line x1="21" y1="20" x2="15" y2="20" />
			<line x1="11" y1="20" x2="3" y2="20" />
			<circle cx="7" cy="4" r="2" />
			<circle cx="11" cy="12" r="2" />
			<circle cx="15" cy="20" r="2" />
		</>
	),
	suitcase: () => (
		<>
			<rect x="2" y="7" width="20" height="14" rx="2" />
			<path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2" />
		</>
	),
	trash: () => (
		<>
			<polyline points="3 6 5 6 21 6" />
			<path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />
			<path d="M10 11v6" />
			<path d="M14 11v6" />
			<path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
		</>
	),
	user: () => (
		<>
			<path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
			<circle cx="12" cy="7" r="4" />
		</>
	),
	users: () => (
		<>
			<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
			<circle cx="9" cy="7" r="4" />
			<path d="M23 21v-2a4 4 0 0 0-3-3.87" />
			<path d="M16 3.13a4 4 0 0 1 0 7.75" />
		</>
	),
	x: () => (
		<>
			<line x1="18" y1="6" x2="6" y2="18" />
			<line x1="6" y1="6" x2="18" y2="18" />
		</>
	),
	zoomIn: () => (
		<>
			<circle cx="11" cy="11" r="8" />
			<line x1="21" y1="21" x2="16.65" y2="16.65" />
			<line x1="11" y1="8" x2="11" y2="14" />
			<line x1="8" y1="11" x2="14" y2="11" />
		</>
	),
} satisfies Record<string, () => JSX.Element>;

export type IconName = keyof typeof ICONS;

export function Icon(props: {
	name: IconName;
	size?: number;
	strokeWidth?: number;
	class?: string;
	style?: JSX.CSSProperties;
}) {
	return (
		<svg
			aria-hidden="true"
			class={props.class}
			style={props.style}
			width={props.size ?? 16}
			height={props.size ?? 16}
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width={props.strokeWidth ?? 2}
			stroke-linecap="round"
			stroke-linejoin="round"
		>
			{ICONS[props.name]()}
		</svg>
	);
}
