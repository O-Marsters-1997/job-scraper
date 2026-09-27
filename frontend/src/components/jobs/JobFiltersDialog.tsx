import {
	CloseButton,
	Content,
	Overlay,
	Portal,
	Root,
	Title,
} from "@kobalte/core/dialog";
import { For, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
	Switch,
	SwitchControl,
	SwitchLabel,
	SwitchThumb,
} from "@/components/ui/switch";
import { DEFAULT_FILTERS, type JobFilters } from "@/lib/jobFilters";
import { cn } from "@/lib/utils";

interface JobFiltersDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	filters: JobFilters;
	onChange: (patch: Partial<JobFilters>) => void;
	sourceOptions: string[];
}

const WORK_OPTIONS = [
	{ value: "remote", label: "Remote" },
	{ value: "hybrid", label: "Hybrid" },
	{ value: "onsite", label: "On-site" },
];

const toggle = <T,>(arr: T[], val: T): T[] =>
	arr.includes(val) ? arr.filter((x) => x !== val) : [...arr, val];

function FilterChip(props: {
	label: string;
	active: boolean;
	onClick: () => void;
}) {
	return (
		<button
			type="button"
			onClick={props.onClick}
			class={cn(
				"rounded-full border px-3 py-1 text-xs font-medium transition-colors",
				props.active
					? "border-primary bg-primary text-primary-foreground"
					: "border-border bg-surface text-muted hover:border-border-strong hover:text-foreground",
			)}
		>
			{props.label}
		</button>
	);
}

function numInput(raw: string): number | undefined {
	const n = Number(raw);
	return raw.trim() !== "" && Number.isFinite(n) ? n : undefined;
}

export function JobFiltersDialog(props: JobFiltersDialogProps) {
	return (
		<Root open={props.open} onOpenChange={props.onOpenChange} modal>
			<Portal>
				<Overlay class="fixed inset-0 z-50 bg-black/30 backdrop-blur-[2px] data-[expanded]:animate-in data-[closed]:animate-out data-[closed]:fade-out-0 data-[expanded]:fade-in-0" />
				<Content
					class={cn(
						"fixed inset-x-0 bottom-0 z-50 flex max-h-[85vh] flex-col rounded-t-2xl border-t border-border bg-surface shadow-2xl",
						"data-[expanded]:animate-in data-[closed]:animate-out",
						"data-[expanded]:slide-in-from-bottom data-[closed]:slide-out-to-bottom",
						"data-[expanded]:fade-in-0 data-[closed]:fade-out-0",
						"data-[expanded]:duration-300 data-[closed]:duration-200",
						"md:inset-x-auto md:bottom-auto md:left-1/2 md:top-1/2 md:w-full md:max-w-md md:-translate-x-1/2 md:-translate-y-1/2 md:rounded-2xl md:border",
						"md:data-[expanded]:slide-in-from-bottom-0 md:data-[expanded]:zoom-in-95 md:data-[closed]:zoom-out-95",
					)}
				>
					<div class="flex items-center justify-between border-b border-border px-5 py-4">
						<Title class="text-sm font-semibold text-foreground">Filters</Title>
						<CloseButton class="flex size-7 items-center justify-center rounded-md text-muted transition-colors hover:bg-surface-muted hover:text-foreground">
							<svg
								width="14"
								height="14"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
								aria-hidden="true"
							>
								<line x1="18" y1="6" x2="6" y2="18" />
								<line x1="6" y1="6" x2="18" y2="18" />
							</svg>
						</CloseButton>
					</div>

					<div class="flex-1 overflow-y-auto px-5 py-4">
						<div class="flex flex-col gap-5">
							<div class="flex flex-col gap-1.5">
								<label class="text-xs font-medium text-muted" for="f-suit">
									Suitability ≥
								</label>
								<Input
									id="f-suit"
									type="number"
									min="0"
									max="100"
									placeholder="—"
									value={props.filters.suit ?? ""}
									onInput={(e) =>
										props.onChange({ suit: numInput(e.currentTarget.value) })
									}
								/>
							</div>

							<Show when={props.sourceOptions.length > 0}>
								<div class="flex flex-col gap-2">
									<p class="text-xs font-medium text-muted">Source</p>
									<div class="flex flex-wrap gap-1.5">
										<For each={props.sourceOptions}>
											{(src) => (
												<FilterChip
													label={src}
													active={props.filters.src.includes(src)}
													onClick={() =>
														props.onChange({
															src: toggle(props.filters.src, src),
														})
													}
												/>
											)}
										</For>
									</div>
								</div>
							</Show>

							<div class="flex flex-col gap-2">
								<p class="text-xs font-medium text-muted">Work arrangement</p>
								<div class="flex flex-wrap gap-1.5">
									<For each={WORK_OPTIONS}>
										{(opt) => (
											<FilterChip
												label={opt.label}
												active={props.filters.work.includes(opt.value)}
												onClick={() =>
													props.onChange({
														work: toggle(props.filters.work, opt.value),
													})
												}
											/>
										)}
									</For>
								</div>
							</div>

							<div class="flex flex-col gap-2">
								<Switch
									checked={props.filters.sal}
									onChange={(v) => props.onChange({ sal: v })}
								>
									<div class="flex items-center gap-2">
										<SwitchControl>
											<SwitchThumb />
										</SwitchControl>
										<SwitchLabel>Salary range</SwitchLabel>
									</div>
								</Switch>
								<Show when={props.filters.sal}>
									<div class="grid grid-cols-2 gap-3">
										<div class="flex flex-col gap-1.5">
											<label
												class="text-xs font-medium text-muted"
												for="f-sal-min"
											>
												Min (£)
											</label>
											<Input
												id="f-sal-min"
												type="number"
												min="0"
												placeholder="0"
												value={props.filters.salMin ?? ""}
												onInput={(e) =>
													props.onChange({
														salMin: numInput(e.currentTarget.value),
													})
												}
											/>
										</div>
										<div class="flex flex-col gap-1.5">
											<label
												class="text-xs font-medium text-muted"
												for="f-sal-max"
											>
												Max (£)
											</label>
											<Input
												id="f-sal-max"
												type="number"
												min="0"
												placeholder="∞"
												value={props.filters.salMax ?? ""}
												onInput={(e) =>
													props.onChange({
														salMax: numInput(e.currentTarget.value),
													})
												}
											/>
										</div>
									</div>
								</Show>
							</div>

							<Switch
								checked={props.filters.showHidden}
								onChange={(v) => props.onChange({ showHidden: v })}
							>
								<div class="flex items-center gap-2">
									<SwitchControl>
										<SwitchThumb />
									</SwitchControl>
									<SwitchLabel>Show hidden jobs</SwitchLabel>
								</div>
							</Switch>
						</div>
					</div>

					<div class="flex items-center justify-between border-t border-border px-5 py-3">
						<Button
							variant="ghost"
							size="sm"
							onClick={() => props.onChange({ ...DEFAULT_FILTERS })}
						>
							Clear all
						</Button>
						<Button size="sm" onClick={() => props.onOpenChange(false)}>
							Done
						</Button>
					</div>
				</Content>
			</Portal>
		</Root>
	);
}
