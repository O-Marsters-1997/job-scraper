import { createSignal, For } from "solid-js";
import { Kbd } from "@/components/Kbd";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";

export type ShortcutRow = readonly [keys: readonly string[], label: string];

export function TableShortcuts(props: {
	noun: string;
	extra?: readonly ShortcutRow[];
}) {
	const [open, setOpen] = createSignal(false);
	const rows = (): ShortcutRow[] => [
		[["j", "↓"], "Next row"],
		[["k", "↑"], "Previous row"],
		[["⇧ ↓"], "Down 5 rows"],
		[["⇧ ↑"], "Up 5 rows"],
		[["Enter"], `Open ${props.noun} in a new tab`],
		...(props.extra ?? []),
		[["/"], "Search"],
		[["]", "→"], "Next page (→ needs an active row)"],
		[["[", "←"], "Previous page (← needs an active row)"],
	];
	return (
		<>
			<div class="flex justify-end">
				<Button variant="ghost" size="sm" onClick={() => setOpen(true)}>
					Shortcuts
				</Button>
			</div>
			<Dialog open={open()} onOpenChange={setOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Keyboard shortcuts</DialogTitle>
					</DialogHeader>
					<dl class="flex flex-col gap-2 text-sm">
						<For each={rows()}>
							{([keys, label]) => (
								<div class="flex items-center justify-between gap-4">
									<dt class="text-muted">{label}</dt>
									<dd class="flex gap-1">
										<For each={keys}>{(k) => <Kbd>{k}</Kbd>}</For>
									</dd>
								</div>
							)}
						</For>
					</dl>
				</DialogContent>
			</Dialog>
		</>
	);
}
