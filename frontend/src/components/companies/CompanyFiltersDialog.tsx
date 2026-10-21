import {
	CloseButton,
	Content,
	Overlay,
	Portal,
	Root,
	Title,
} from "@kobalte/core/dialog";
import { For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { ToggleChip } from "@/components/ToggleChip";
import { Button } from "@/components/ui/button";
import type { CompanyListState } from "@/lib/companyFilters";
import { cn } from "@/lib/utils";
import type { ATSCount } from "@/types/company";

interface CompanyFiltersDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	ats: string[];
	atsCounts: ATSCount[];
	onChange: (patch: Partial<CompanyListState>) => void;
}

const toggle = (arr: string[], val: string): string[] =>
	arr.includes(val) ? arr.filter((x) => x !== val) : [...arr, val];

export function CompanyFiltersDialog(props: CompanyFiltersDialogProps) {
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
							<Icon name="x" size={14} />
						</CloseButton>
					</div>

					<div class="flex-1 overflow-y-auto px-5 py-4">
						<Show when={props.atsCounts.length > 0}>
							<div class="flex flex-col gap-2">
								<p class="text-xs font-medium text-muted">ATS</p>
								<div class="flex flex-wrap gap-1.5">
									<For each={props.atsCounts}>
										{(opt) => (
											<ToggleChip
												solid
												active={props.ats.includes(opt.source)}
												onClick={() =>
													props.onChange({ ats: toggle(props.ats, opt.source) })
												}
											>
												{opt.source}
												<span class="font-mono tabular-nums opacity-60">
													{opt.companies}
												</span>
											</ToggleChip>
										)}
									</For>
								</div>
							</div>
						</Show>
					</div>

					<div class="flex items-center justify-between border-t border-border px-5 py-3">
						<Button
							variant="ghost"
							size="sm"
							onClick={() => props.onChange({ ats: [] })}
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
