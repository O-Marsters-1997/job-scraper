import { Link } from "@tanstack/solid-router";
import { Show } from "solid-js";

interface ErrorStateProps {
	title?: string;
	message?: string;
	error?: unknown;
	onRetry?: () => void;
}

function errorDetail(error: unknown): string | null {
	if (!error) return null;
	if (error instanceof Error) return error.message;
	if (typeof error === "string") return error;
	return null;
}

// Branded fallback for thrown render/loader errors.
export function ErrorState(props: ErrorStateProps) {
	const detail = () => errorDetail(props.error);

	return (
		<div class="flex min-h-[60vh] flex-col items-center justify-center px-6 py-12 text-center">
			<div class="flex h-11 w-11 items-center justify-center rounded-full border border-destructive/30 bg-destructive-subtle text-destructive-strong">
				<svg
					aria-hidden="true"
					width="20"
					height="20"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<path d="M12 9v4" />
					<path d="M12 17h.01" />
					<path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0Z" />
				</svg>
			</div>
			<h1 class="mt-5 text-lg font-bold tracking-tight text-foreground">
				{props.title ?? "Something went wrong"}
			</h1>
			<p class="mt-1.5 max-w-sm text-sm text-muted">
				{props.message ??
					"This page hit an unexpected error. Try again, or head back to your jobs."}
			</p>
			<Show when={detail()}>
				{(d) => (
					<p class="mt-3 max-w-md break-words font-mono text-xs text-faint">
						{d()}
					</p>
				)}
			</Show>
			<div class="mt-6 flex items-center gap-2">
				<Show when={props.onRetry}>
					<button
						type="button"
						onClick={() => props.onRetry?.()}
						class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition hover:bg-primary-hover"
					>
						Try again
					</button>
				</Show>
				<Link
					to="/jobs"
					class="inline-flex h-9 items-center rounded-md border border-border bg-surface px-4 text-sm font-medium text-muted transition hover:border-border-strong hover:text-foreground"
				>
					Back to jobs
				</Link>
			</div>
		</div>
	);
}

// Branded 404 for unmatched routes.
export function NotFoundState() {
	return (
		<div class="relative flex min-h-[60vh] flex-col items-center justify-center overflow-hidden px-6 py-12 text-center">
			<div class="brand-aurora-shell" aria-hidden="true">
				<span class="brand-ribbon brand-ribbon-center" />
			</div>
			<div class="relative z-10 flex flex-col items-center">
				<span class="font-mono text-2xl font-bold tracking-tight text-faint">
					404
				</span>
				<h1 class="mt-3 text-lg font-bold tracking-tight text-foreground">
					Page not found
				</h1>
				<p class="mt-1.5 max-w-sm text-sm text-muted">
					That page doesn't exist or has moved.
				</p>
				<Link
					to="/jobs"
					class="mt-6 inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition hover:bg-primary-hover"
				>
					Back to jobs
				</Link>
			</div>
		</div>
	);
}
