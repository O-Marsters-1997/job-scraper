import { createEffect, createSignal, Show } from "solid-js";
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
	steerOpen: boolean;
	onSteerClose: () => void;
	error?: string;
	onRedraft: (steer?: string) => void;
	onRevert: () => void;
}) {
	const [steer, setSteer] = createSignal("");
	const [attempt, setAttempt] = createSignal<{ from: string; to?: string }>();
	const differs = () => props.text !== props.base;
	const undoable = () => {
		const a = attempt();
		return a?.to !== undefined && a.to !== a.from && props.text === a.to;
	};
	const closeIfIdle = () => {
		if (!steer().trim() && !props.pending) props.onSteerClose();
	};
	const redraft = () => {
		setAttempt({ from: props.text });
		props.onRedraft(steer().trim() || undefined);
	};
	createEffect(() => {
		const a = attempt();
		if (!a || a.to !== undefined) return;
		if (props.pending || props.error || props.text === a.from) return;
		setAttempt({ ...a, to: props.text });
	});
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
			<Show when={props.steerOpen}>
				<div
					class="flex flex-col gap-3"
					onFocusOut={(e) => {
						if (!e.currentTarget.contains(e.relatedTarget as Node | null))
							closeIfIdle();
					}}
				>
					<Input
						aria-label="Steer the redraft"
						placeholder="Steer it, e.g. lead with platform work"
						maxLength={MAX_STEER}
						value={steer()}
						disabled={props.pending}
						onInput={(e) => setSteer(e.currentTarget.value)}
						onKeyDown={(e) => {
							if (e.key === "Enter" && !props.pending) redraft();
							if (e.key === "Escape") closeIfIdle();
						}}
					/>
					<Button size="sm" loading={props.pending} onClick={redraft}>
						<Icon name="wand" size={14} />
						Redraft
					</Button>
				</div>
			</Show>
			<Show when={props.error}>
				{(message) => (
					<p role="alert" class="text-xs text-destructive">
						{message()}
					</p>
				)}
			</Show>
			<div class="flex items-center gap-2">
				<Show when={undoable() || differs()}>
					<Button
						variant="ghost"
						size="sm"
						disabled={props.pending}
						onClick={() => props.onRevert()}
					>
						<Icon name="rotateCcw" size={14} />
						{undoable() ? "Undo" : "Revert to base"}
					</Button>
				</Show>
			</div>
		</section>
	);
}
