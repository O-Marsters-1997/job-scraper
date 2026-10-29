import { Link, useNavigate } from "@tanstack/solid-router";
import {
	createEffect,
	createMemo,
	createSignal,
	For,
	type JSX,
	Show,
} from "solid-js";
import { Kbd } from "@/components/Kbd";
import { useMoveApplication } from "@/hooks/useApplications";
import { useShortcuts } from "@/hooks/useShortcuts";
import { formatChaseDate, isChaseOverdue } from "@/lib/chase";
import { CV_STAGE_LABEL, cvStage } from "@/lib/cvStage";
import { assignStatusKeys, BOARD_NAV_KEYS } from "@/lib/statusKeys";
import { cn } from "@/lib/utils";
import type { ApplicationWithDetails } from "../../../types/application";
import type { ApplicationStatus } from "../../../types/applicationStatus";

function AppCard(props: {
	app: ApplicationWithDetails;
	dragging: boolean;
	selected: boolean;
	onSelect: () => void;
	onDragStart: () => void;
	onDragEnd: () => void;
}) {
	return (
		<li
			draggable={true}
			data-app-id={props.app.ID}
			onDragStart={(e) => {
				e.dataTransfer?.setData("text/plain", props.app.ID);
				props.onDragStart();
			}}
			onDragEnd={() => props.onDragEnd()}
			class={cn(
				"relative cursor-grab rounded-lg border border-border bg-surface p-3 shadow-sm transition active:cursor-grabbing",
				props.dragging && "opacity-40",
				props.selected && "ring-2 ring-primary",
			)}
		>
			<button
				type="button"
				aria-label={`Select ${props.app.JobTitle}`}
				aria-pressed={props.selected}
				onClick={() => props.onSelect()}
				class="absolute inset-0 rounded-lg"
			/>
			<Link
				to="/applications/$id"
				params={{ id: props.app.ID }}
				class="relative line-clamp-2 text-sm font-medium text-foreground hover:underline"
			>
				{props.app.JobTitle}
			</Link>
			<p class="pointer-events-none relative truncate text-xs text-faint">
				{props.app.JobCompanySlug}
			</p>
			<div class="pointer-events-none relative mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted">
				<span>{CV_STAGE_LABEL[cvStage(props.app.Draft)]}</span>
				<Show when={props.app.ChaseBy}>
					{(c) => (
						<span
							class={cn(
								"font-mono tabular-nums",
								isChaseOverdue(c(), new Date()) && "text-destructive-strong",
							)}
						>
							Chase {formatChaseDate(c())}
						</span>
					)}
				</Show>
				<Show when={props.app.JobClosedAt}>
					<span class="text-faint">Closed</span>
				</Show>
			</div>
		</li>
	);
}

function Dot(props: { colour: string }) {
	return (
		<span
			class="inline-block size-2 shrink-0 rounded-full"
			style={{ background: props.colour }}
		/>
	);
}

export function ApplicationsBoard(props: {
	apps: ApplicationWithDetails[];
	statuses: ApplicationStatus[];
}): JSX.Element {
	const move = useMoveApplication();
	const navigate = useNavigate();
	const statusKeys = createMemo(() => assignStatusKeys(props.statuses));
	const [collapsed, setCollapsed] = createSignal<Record<string, boolean>>({});
	const [dragging, setDragging] = createSignal<string>();
	const [over, setOver] = createSignal<string>();
	const [selectedID, setSelectedID] = createSignal<string>();

	const toggle = (id: string) => setCollapsed((c) => ({ ...c, [id]: !c[id] }));
	const inStatus = (id: string) => props.apps.filter((a) => a.StatusID === id);
	const selected = () => props.apps.find((a) => a.ID === selectedID());
	const moveTo = (app: ApplicationWithDetails, statusID: string) => {
		if (app.StatusID === statusID) return;
		setCollapsed((c) => ({ ...c, [statusID]: false }));
		move.mutate({ id: app.ID, data: { status_id: statusID } });
	};
	const step = (delta: number) => {
		const app = selected();
		if (!app) {
			const first = props.statuses
				.filter((s) => !collapsed()[s.ID])
				.flatMap((s) => inStatus(s.ID))[0];
			if (first) setSelectedID(first.ID);
			return;
		}
		const column = inStatus(app.StatusID);
		const next = column[column.findIndex((a) => a.ID === app.ID) + delta];
		if (next) setSelectedID(next.ID);
	};
	const shift = (delta: number) => {
		const app = selected();
		if (!app) return false;
		const i = props.statuses.findIndex((s) => s.ID === app.StatusID);
		const target = props.statuses[i + delta];
		if (target) moveTo(app, target.ID);
	};
	const byLetter = (e: KeyboardEvent) => {
		const app = selected();
		const target = props.statuses.find((s) => statusKeys()[s.ID] === e.key);
		if (!app || !target) return false;
		moveTo(app, target.ID);
	};
	const letters = Object.fromEntries(
		[..."abcdefghijklmnopqrstuvwxyz"]
			.filter((c) => !BOARD_NAV_KEYS.includes(c))
			.map((c) => [c, byLetter]),
	);
	useShortcuts({
		...letters,
		j: () => step(1),
		ArrowDown: () => step(1),
		k: () => step(-1),
		ArrowUp: () => step(-1),
		h: () => shift(-1),
		ArrowLeft: () => shift(-1),
		l: () => shift(1),
		ArrowRight: () => shift(1),
		Enter: (e) => {
			const app = selected();
			if (
				!app ||
				(e.target as Element).closest("a, button:not([aria-pressed])")
			)
				return false;
			navigate({ to: "/applications/$id", params: { id: app.ID } });
		},
		Escape: () => (selected() ? setSelectedID(undefined) : false),
		"/": () => {
			const input = document.getElementById("applications-search");
			if (!(input instanceof HTMLInputElement)) return false;
			input.focus();
			input.select();
		},
	});
	createEffect(() => {
		const id = selectedID();
		if (!id) return;
		document
			.querySelector(`[data-app-id="${id}"]`)
			?.scrollIntoView({ block: "nearest", inline: "nearest" });
	});

	const endDrag = () => {
		setDragging(undefined);
		setOver(undefined);
	};
	const dropTarget = (statusID: string) => ({
		onDragOver: (e: DragEvent) => {
			e.preventDefault();
			setOver(statusID);
		},
		onDragLeave: (e: DragEvent) => {
			if (
				!(e.currentTarget as Element).contains(e.relatedTarget as Node | null)
			)
				setOver(undefined);
		},
		onDrop: (e: DragEvent) => {
			e.preventDefault();
			const id = dragging();
			const app = props.apps.find((a) => a.ID === id);
			if (id && app && app.StatusID !== statusID)
				move.mutate({ id, data: { status_id: statusID } });
			endDrag();
		},
	});

	return (
		<>
			<p class="mb-2 flex flex-wrap items-center gap-x-2 text-xs text-muted">
				<span class="truncate font-medium text-foreground">
					{selected()?.JobTitle ?? "No job selected"}
				</span>
				<Kbd>j</Kbd>
				<Kbd>k</Kbd>
				select
				<Kbd>h</Kbd>
				<Kbd>l</Kbd>
				move
				<Kbd>Enter</Kbd>
				open
				<Kbd>Esc</Kbd>
				clear
				<Kbd>/</Kbd>
				search
			</p>
			<div class="scroll-slim flex h-[calc(100vh-17rem)] gap-2 overflow-x-auto pb-1">
				<For each={props.statuses}>
					{(s) => (
						<Show
							when={!collapsed()[s.ID]}
							fallback={
								<button
									type="button"
									aria-label={`Expand ${s.Name}`}
									{...dropTarget(s.ID)}
									onClick={() => toggle(s.ID)}
									class={cn(
										"flex w-10 shrink-0 flex-col items-center gap-3 rounded-xl border border-border bg-surface py-3 transition hover:bg-surface-muted",
										dragging() && "border-dashed border-primary",
										over() === s.ID && "ring-2 ring-primary",
									)}
								>
									<Dot colour={s.Colour} />
									<Show when={statusKeys()[s.ID]}>
										{(k) => <Kbd>{k()}</Kbd>}
									</Show>
									<span class="font-mono text-xs tabular-nums text-muted">
										{inStatus(s.ID).length}
									</span>
									<span class="text-xs font-medium [writing-mode:vertical-rl]">
										{s.Name}
									</span>
								</button>
							}
						>
							<section
								{...dropTarget(s.ID)}
								class={cn(
									"flex min-w-60 flex-1 flex-col rounded-xl bg-surface-muted p-2 transition",
									over() === s.ID && "ring-2 ring-primary",
								)}
							>
								<header class="flex items-center gap-2 px-1 pb-2 text-sm font-medium">
									<Dot colour={s.Colour} />
									{s.Name}
									<Show when={statusKeys()[s.ID]}>
										{(k) => <Kbd>{k()}</Kbd>}
									</Show>
									<span class="font-mono text-xs text-faint tabular-nums">
										{inStatus(s.ID).length}
									</span>
									<button
										type="button"
										aria-label={`Collapse ${s.Name}`}
										onClick={() => toggle(s.ID)}
										class="ml-auto rounded px-1.5 text-faint hover:bg-surface hover:text-foreground"
									>
										⇤
									</button>
								</header>
								<ul class="scroll-slim flex min-h-24 flex-1 flex-col gap-2 overflow-y-auto">
									<For each={inStatus(s.ID)}>
										{(app) => (
											<AppCard
												app={app}
												dragging={dragging() === app.ID}
												selected={selectedID() === app.ID}
												onSelect={() => setSelectedID(app.ID)}
												onDragStart={() => setDragging(app.ID)}
												onDragEnd={endDrag}
											/>
										)}
									</For>
								</ul>
							</section>
						</Show>
					)}
				</For>
			</div>
		</>
	);
}
