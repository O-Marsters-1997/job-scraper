import { createMemo, createSignal, For, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { parseReply } from "@/lib/promptHandoff";
import { cn } from "@/lib/utils";
import { wordDiff } from "@/lib/wordDiff";

function Diff(props: { from: string; to: string }) {
	return (
		<For each={wordDiff(props.from, props.to)}>
			{(part) => (
				<span
					class={cn(
						part.op === "del" &&
							"text-faint line-through decoration-destructive/60",
						part.op === "add" && "rounded-sm bg-accent-subtle text-accent-text",
					)}
				>
					{part.text}
				</span>
			)}
		</For>
	);
}

export function PasteReplyDialog(props: {
	open: boolean;
	texts: Record<string, string>;
	onOpenChange: (open: boolean) => void;
	onApply: (changes: { path: string; to: string }[]) => void;
}) {
	const [reply, setReply] = createSignal("");
	const parsed = createMemo(() => parseReply(reply(), props.texts));
	const count = () => parsed().changes.length;
	const close = () => {
		setReply("");
		props.onOpenChange(false);
	};
	return (
		<Dialog open={props.open} onOpenChange={(o) => !o && close()}>
			<DialogContent class="max-w-2xl">
				<DialogHeader>
					<DialogTitle>Paste Claude Code's reply</DialogTitle>
					<p class="text-sm text-muted">
						Changes land in the sheet unsaved, so you can Try them first.
					</p>
				</DialogHeader>
				<Textarea
					rows={7}
					class="font-mono text-xs"
					aria-label="Claude Code's reply"
					placeholder={
						'Paste the whole reply. FastTrack picks out the <file path="…"> blocks.'
					}
					value={reply()}
					onInput={(e) => setReply(e.currentTarget.value)}
				/>
				<Show when={reply().trim()}>
					<Show
						when={!parsed().empty}
						fallback={
							<p class="text-xs text-destructive-strong">
								No &lt;file path="…"&gt; blocks found. The tailoring prompt asks
								Claude Code to reply in that format; ask it to try again.
							</p>
						}
					>
						<div class="flex max-h-72 flex-col gap-2 overflow-y-auto">
							<For
								each={parsed().changes}
								fallback={
									<p class="text-xs text-muted">
										The reply matches your files already. Nothing to apply.
									</p>
								}
							>
								{(c) => (
									<div class="rounded-lg border border-border bg-surface-muted p-3">
										<p class="mb-1 font-mono text-xs font-medium text-foreground">
											{c.path}
										</p>
										<pre class="whitespace-pre-wrap font-mono text-xs leading-relaxed text-muted">
											<Diff from={c.from} to={c.to} />
										</pre>
									</div>
								)}
							</For>
							<Show when={parsed().skipped.length > 0}>
								<p class="text-xs text-status-interview">
									Skipped {parsed().skipped.join(", ")}: only the editable
									prompt files can change.
								</p>
							</Show>
						</div>
					</Show>
				</Show>
				<DialogFooter>
					<Button variant="ghost" onClick={close}>
						Cancel
					</Button>
					<Button
						disabled={count() === 0}
						onClick={() => {
							props.onApply(parsed().changes);
							close();
						}}
					>
						{count() > 0
							? `Apply ${count()} file${count() === 1 ? "" : "s"}`
							: "Apply"}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
