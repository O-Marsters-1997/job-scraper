import { Link } from "@tanstack/solid-router";
import { createSignal, For, type JSX, Show } from "solid-js";
import { useMoveApplication } from "@/hooks/useApplications";
import { formatChaseDate, isChaseOverdue } from "@/lib/chase";
import { CV_STAGE_LABEL, cvStage } from "@/lib/cvStage";
import { cn } from "@/lib/utils";
import type { ApplicationWithDetails } from "../../../types/application";
import type { ApplicationStatus } from "../../../types/applicationStatus";

function AppCard(props: {
	app: ApplicationWithDetails;
	dragging: boolean;
	onDragStart: () => void;
	onDragEnd: () => void;
}) {
	return (
		<li
			draggable={true}
			onDragStart={(e) => {
				e.dataTransfer?.setData("text/plain", props.app.ID);
				props.onDragStart();
			}}
			onDragEnd={() => props.onDragEnd()}
			class={cn(
				"cursor-grab rounded-lg border border-border bg-surface p-3 shadow-sm transition active:cursor-grabbing",
				props.dragging && "opacity-40",
			)}
		>
			<Link
				to="/applications/$id"
				params={{ id: props.app.ID }}
				class="line-clamp-2 text-sm font-medium text-foreground hover:underline"
			>
				{props.app.JobTitle}
			</Link>
			<p class="truncate text-xs text-faint">{props.app.JobCompanySlug}</p>
			<div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted">
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
	const [collapsed, setCollapsed] = createSignal<Record<string, boolean>>({});
	const [dragging, setDragging] = createSignal<string>();
	const [over, setOver] = createSignal<string>();

	const toggle = (id: string) => setCollapsed((c) => ({ ...c, [id]: !c[id] }));
	const inStatus = (id: string) => props.apps.filter((a) => a.StatusID === id);
	const endDrag = () => {
		setDragging(undefined);
		setOver(undefined);
	};
	const dropTarget = (statusID: string) => ({
		onDragOver: (e: DragEvent) => {
			e.preventDefault();
			setOver(statusID);
		},
		onDragLeave: () => setOver(undefined),
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
		<div class="scroll-slim flex h-[calc(100vh-15rem)] gap-2 overflow-x-auto pb-1">
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
	);
}
