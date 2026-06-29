import { Link } from "@tanstack/solid-router";
import type { JSX } from "solid-js";
import { Label } from "@/components/ui/label";
import {
	Sheet,
	SheetClose,
	SheetContent,
	SheetHeader,
	SheetTitle,
	SheetTrigger,
} from "@/components/ui/sheet";
import { Switch, SwitchControl, SwitchThumb } from "@/components/ui/switch";
import { demoDataEnabled, setDemoData } from "@/lib/demoData";

export default function SettingsPanel() {
	return (
		<Sheet>
			<SheetTrigger
				class="flex size-8 items-center justify-center rounded-full border border-accent-border bg-accent-subtle text-accent-text transition-opacity hover:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2"
				aria-label="Open settings"
			>
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
					<path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
					<circle cx="12" cy="7" r="4" />
				</svg>
			</SheetTrigger>

			<SheetContent>
				<SheetHeader>
					<SheetTitle>Settings</SheetTitle>
					<SheetClose
						class="flex size-7 items-center justify-center rounded-md text-muted transition-colors hover:bg-surface-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
						aria-label="Close settings"
					>
						<svg
							aria-hidden="true"
							width="14"
							height="14"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
						>
							<line x1="18" y1="6" x2="6" y2="18" />
							<line x1="6" y1="6" x2="18" y2="18" />
						</svg>
					</SheetClose>
				</SheetHeader>

				<div class="flex flex-col gap-6 px-6 py-5">
					{/* Account / persona settings */}
					<div class="flex flex-col gap-1">
						<p class="mb-2 text-2xs font-semibold uppercase tracking-wider text-faint">
							Account
						</p>
						<SettingsNavRow
							to="/settings/profile"
							label="Profile"
							description="Your job-seeking preferences and CV context"
						>
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
						</SettingsNavRow>
						<SettingsNavRow
							to="/settings/scoring"
							label="Scoring"
							description="Weights and criteria for job suitability scores"
						>
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
						</SettingsNavRow>
						<SettingsNavRow
							to="/settings/ai"
							label="AI"
							description="Model, API key, and inference preferences"
						>
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
						</SettingsNavRow>
					</div>

					<div class="border-t border-border" />

					{/* Session toggle */}
					<SettingRow
						label="Preview demo data"
						description="Show seeded sample data for this session only."
					>
						<Switch checked={demoDataEnabled()} onChange={setDemoData}>
							<SwitchControl>
								<SwitchThumb />
							</SwitchControl>
						</Switch>
					</SettingRow>
				</div>
			</SheetContent>
		</Sheet>
	);
}

/** A nav link row that navigates and closes the drawer in one action. */
function SettingsNavRow(props: {
	to: string;
	label: string;
	description: string;
	children: JSX.Element;
}) {
	return (
		<SheetClose
			as={Link}
			to={props.to}
			class="flex items-center gap-3 rounded-md px-2 py-2 text-sm text-foreground transition-colors hover:bg-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
		>
			<span class="flex size-7 shrink-0 items-center justify-center rounded-md border border-border bg-surface-muted text-muted">
				{props.children}
			</span>
			<div class="flex flex-col gap-0.5">
				<span class="font-medium leading-none">{props.label}</span>
				<span class="text-xs text-faint">{props.description}</span>
			</div>
			<svg
				aria-hidden="true"
				class="ml-auto shrink-0 text-faint"
				width="14"
				height="14"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			>
				<polyline points="9 18 15 12 9 6" />
			</svg>
		</SheetClose>
	);
}

function SettingRow(props: {
	label: string;
	description: string;
	children: JSX.Element;
}) {
	return (
		<div class="flex items-start justify-between gap-4">
			<div class="flex flex-col gap-0.5">
				<Label>{props.label}</Label>
				<p class="text-xs text-faint">{props.description}</p>
			</div>
			{props.children}
		</div>
	);
}
