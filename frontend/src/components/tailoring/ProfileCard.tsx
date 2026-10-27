import { createSignal, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { EditDiff } from "./MarginCard";

const MAX_STEER = 200;

export function ProfileCard(props: {
	base: string;
	text: string;
	top: number;
	pending: boolean;
	error?: string;
	onRedraft: (steer?: string) => void;
	onRevert: () => void;
}) {
	const [steer, setSteer] = createSignal("");
	const [before, setBefore] = createSignal<string>();
	const differs = () => props.text !== props.base;
	const undoable = () => {
		const prior = before();
		return prior !== undefined && props.text !== prior;
	};
	const redraft = () => {
		setBefore(props.text);
		props.onRedraft(steer().trim() || undefined);
	};
	const revert = () => {
		setBefore(undefined);
		props.onRevert();
	};
	return (
		<section
			aria-label="Profile"
			data-testid="profile-card"
			class="absolute inset-x-0 flex flex-col gap-3 rounded-xl bg-surface px-3.5 py-3 text-sm ring-1 ring-border"
			style={{ top: `${props.top}px` }}
		>
			<p class="text-xs font-medium text-foreground">Profile</p>
			<Show when={differs()}>
				<EditDiff from={props.base} to={props.text} title="Against your base" />
			</Show>
			<Input
				aria-label="Steer the redraft"
				placeholder="Steer it, e.g. lead with platform work"
				maxLength={MAX_STEER}
				value={steer()}
				disabled={props.pending}
				onInput={(e) => setSteer(e.currentTarget.value)}
				onKeyDown={(e) => {
					if (e.key === "Enter" && !props.pending) redraft();
				}}
			/>
			<Show when={props.error}>
				{(message) => (
					<p role="alert" class="text-xs text-destructive">
						{message()}
					</p>
				)}
			</Show>
			<div class="flex items-center gap-2">
				<Button size="sm" loading={props.pending} onClick={redraft}>
					<Icon name="wand" size={14} />
					Redraft
				</Button>
				<Show
					when={undoable()}
					fallback={
						<Show when={differs()}>
							<Button
								variant="ghost"
								size="sm"
								disabled={props.pending}
								onClick={revert}
							>
								<Icon name="rotateCcw" size={14} />
								Revert to base
							</Button>
						</Show>
					}
				>
					<Button
						variant="ghost"
						size="sm"
						disabled={props.pending}
						onClick={revert}
					>
						<Icon name="rotateCcw" size={14} />
						Undo
					</Button>
				</Show>
			</div>
		</section>
	);
}
