import { onMount, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { SuggestAction } from "@/types/tailoring";

export const ACTION_LABEL: Record<SuggestAction, string> = {
	fit: "Fit to fewer lines",
	tighten: "Tighten",
	ground: "Use only my Achievement",
	verb: "Stronger verb",
	ask: "Ask",
};

function AskForm(props: {
	onSend: (prompt: string) => void;
	onClose: () => void;
}) {
	let input: HTMLInputElement | undefined;
	onMount(() => input?.focus());
	return (
		<form
			class="flex items-center gap-1 rounded-lg bg-sidebar p-1 shadow-xl ring-1 ring-sidebar-border animate-in fade-in zoom-in-95 duration-150"
			onSubmit={(e) => {
				e.preventDefault();
				const prompt = input?.value.trim();
				if (prompt) props.onSend(prompt);
			}}
		>
			<input
				ref={input}
				placeholder="Make it concise, add the metric…"
				aria-label="Ask Haiku to edit this line"
				class="h-8 min-w-0 flex-1 rounded-md bg-sidebar-hover px-2 text-xs text-sidebar-foreground-strong caret-sidebar-active-foreground outline-none placeholder:text-sidebar-foreground"
				onKeyDown={(e) => {
					if (e.key === "Escape") {
						e.stopPropagation();
						props.onClose();
					}
				}}
			/>
			<button
				type="submit"
				aria-label="Send"
				class="grid size-8 shrink-0 place-items-center rounded-md text-sidebar-foreground-strong transition-colors hover:bg-sidebar-hover"
			>
				<Icon name="arrowRight" size={14} />
			</button>
		</form>
	);
}

export function WandMenu(props: {
	top: number;
	canFit: boolean;
	canGround: boolean;
	askOpen: boolean;
	onAskOpen: (open: boolean) => void;
	onAction: (action: SuggestAction, prompt?: string) => void;
}) {
	return (
		<div
			class="absolute right-2 z-20 flex w-[min(20rem,calc(100%-1rem))] justify-end"
			style={{ top: `${props.top}px` }}
		>
			<Show
				when={props.askOpen}
				fallback={
					<DropdownMenu modal={false} placement="bottom-end">
						<DropdownMenuTrigger
							aria-label="Edit with Haiku"
							title="Edit with Haiku (⌘K)"
							class="grid size-8 place-items-center rounded-full bg-surface text-accent-text shadow-md ring-1 ring-border transition-[transform,background-color] duration-150 ease-out hover:scale-110 hover:bg-accent-subtle animate-in fade-in zoom-in-90"
						>
							<Icon name="wand" size={15} />
						</DropdownMenuTrigger>
						<DropdownMenuContent
							onCloseAutoFocus={(e: Event) => {
								if (props.askOpen) e.preventDefault();
							}}
						>
							<Show when={props.canFit}>
								<DropdownMenuItem onSelect={() => props.onAction("fit")}>
									{ACTION_LABEL.fit}
								</DropdownMenuItem>
							</Show>
							<DropdownMenuItem onSelect={() => props.onAction("tighten")}>
								{ACTION_LABEL.tighten}
							</DropdownMenuItem>
							<Show when={props.canGround}>
								<DropdownMenuItem onSelect={() => props.onAction("ground")}>
									{ACTION_LABEL.ground}
								</DropdownMenuItem>
							</Show>
							<DropdownMenuItem onSelect={() => props.onAction("verb")}>
								{ACTION_LABEL.verb}
							</DropdownMenuItem>
							<DropdownMenuSeparator />
							<DropdownMenuItem onSelect={() => props.onAskOpen(true)}>
								{ACTION_LABEL.ask}
								<kbd class="ml-auto font-mono text-2xs text-muted">⌘K</kbd>
							</DropdownMenuItem>
						</DropdownMenuContent>
					</DropdownMenu>
				}
			>
				<AskForm
					onClose={() => props.onAskOpen(false)}
					onSend={(prompt) => {
						props.onAskOpen(false);
						props.onAction("ask", prompt);
					}}
				/>
			</Show>
		</div>
	);
}
