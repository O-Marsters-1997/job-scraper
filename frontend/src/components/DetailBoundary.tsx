import type { Accessor, JSX } from "solid-js";
import { Show } from "solid-js";
import { ErrorState } from "@/components/ErrorState";

type Props<T> = {
	query: {
		isPending: boolean;
		isError: boolean;
		error: unknown;
		refetch: () => unknown;
	};
	data: T | undefined;
	notFound: JSX.Element;
	children: (data: Accessor<NonNullable<T>>) => JSX.Element;
};

export function DetailBoundary<T>(props: Props<T>): JSX.Element {
	return (
		<Show
			when={!props.query.isPending}
			fallback={<div class="px-7 py-6 text-sm text-muted">Loading…</div>}
		>
			<Show
				when={props.data}
				fallback={
					<Show when={props.query.isError} fallback={props.notFound}>
						<ErrorState
							error={props.query.error}
							onRetry={() => props.query.refetch()}
						/>
					</Show>
				}
			>
				{(data) => props.children(data)}
			</Show>
		</Show>
	);
}
