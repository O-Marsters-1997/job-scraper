import { type JSX } from "solid-js";
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
