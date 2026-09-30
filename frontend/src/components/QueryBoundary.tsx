import type { Accessor, JSX } from "solid-js";
import { Show } from "solid-js";
import { ErrorState } from "@/components/ErrorState";
import { SkeletonList } from "@/components/ui/skeleton";

type Props<Q, T> = {
	query: {
		isPending: boolean;
		isError: boolean;
		error: unknown;
		refetch: () => unknown;
		data: Q | undefined;
	};
	select?: (data: Q) => T | undefined;
	notFound?: JSX.Element;
	fallbackRows?: number;
	children: (data: Accessor<NonNullable<T>>) => JSX.Element;
};

export function QueryBoundary<Q, T = Q>(props: Props<Q, T>): JSX.Element {
	const data = () => {
		const d = props.query.data;
		if (d === undefined) return undefined;
		return props.select ? props.select(d) : (d as unknown as T);
	};
	return (
		<Show
			when={data()}
			fallback={
				<Show
					when={props.query.isError}
					fallback={
						<Show
							when={props.notFound && !props.query.isPending}
							fallback={<SkeletonList rows={props.fallbackRows} />}
						>
							{props.notFound}
						</Show>
					}
				>
					<ErrorState
						error={props.query.error}
						onRetry={() => props.query.refetch()}
					/>
				</Show>
			}
		>
			{(d) => props.children(d as Accessor<NonNullable<T>>)}
		</Show>
	);
}
