import { Link, useLocation } from "@tanstack/solid-router";
import { For, type JSX } from "solid-js";
import { Icon } from "@/components/Icon";
import { Label } from "@/components/ui/label";
import {
	Sheet,
	SheetClose,
	SheetContent,
	SheetHeader,
	SheetTitle,
	SheetTrigger,
} from "@/components/ui/sheet";
import {
	Switch,
	SwitchControl,
	SwitchLabel,
	SwitchThumb,
} from "@/components/ui/switch";
import { demoDataEnabled, setDemoData } from "@/lib/demoData";
import {
	SETTINGS_SECTIONS,
	type SettingsSection,
} from "@/lib/settingsSections";
import { cn } from "@/lib/utils";

export default function SettingsPanel() {
	const location = useLocation();

	return (
		<Sheet>
			<SheetTrigger
				class="flex size-8 items-center justify-center rounded-full border border-accent-border bg-accent-subtle text-accent-text transition-opacity hover:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2"
				aria-label="Open settings"
			>
				<Icon name="user" size={15} />
			</SheetTrigger>

			<SheetContent>
				<SheetHeader>
					<SheetTitle>Settings</SheetTitle>
					<SheetClose
						class="flex size-7 items-center justify-center rounded-md text-muted transition-colors hover:bg-surface-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
						aria-label="Close settings"
					>
						<Icon name="x" size={14} />
					</SheetClose>
				</SheetHeader>

				<div class="flex flex-col gap-6 px-6 py-5">
					<SettingRow
						label="Preview demo data"
						description="Show seeded sample data for this session only."
					>
						<Switch checked={demoDataEnabled()} onChange={setDemoData}>
							<SwitchLabel class="sr-only">Preview demo data</SwitchLabel>
							<SwitchControl>
								<SwitchThumb />
							</SwitchControl>
						</Switch>
					</SettingRow>

					<div class="border-t border-border" />

					<For each={SETTINGS_SECTIONS}>
						{(group) => (
							<div class="flex flex-col gap-1">
								<p class="mb-2 text-2xs font-semibold uppercase tracking-wider text-faint">
									{group.heading}
								</p>
								<For each={group.sections}>
									{(section) => (
										<SettingsNavRow
											section={section}
											current={location().pathname === section.to}
										/>
									)}
								</For>
							</div>
						)}
					</For>

					<div class="border-t border-border" />

					<SheetClose
						as={Link}
						to="/settings"
						class="text-sm font-medium text-accent-text transition-colors hover:opacity-80"
					>
						All settings
					</SheetClose>
				</div>
			</SheetContent>
		</Sheet>
	);
}

function SettingsNavRow(props: { section: SettingsSection; current: boolean }) {
	return (
		<SheetClose
			as={Link}
			to={props.section.to}
			class={cn(
				"flex items-center gap-3 rounded-md px-2 py-2 text-sm text-foreground transition-colors hover:bg-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
				props.current && "bg-surface-muted",
			)}
		>
			<span class="flex size-7 shrink-0 items-center justify-center rounded-md border border-border bg-surface-muted text-muted">
				<Icon name={props.section.icon} size={15} />
			</span>
			<div class="flex flex-col gap-0.5">
				<span class="font-medium leading-none">{props.section.label}</span>
				<span class="text-xs text-faint">{props.section.description}</span>
			</div>
			<Icon name="chevronRight" size={14} class="ml-auto shrink-0 text-faint" />
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
