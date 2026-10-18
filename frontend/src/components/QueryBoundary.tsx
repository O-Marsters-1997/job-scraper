import type { Accessor, JSX } from "solid-js";
import { Match, Switch } from "solid-js";
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
		<Switch fallback={<SkeletonList rows={props.fallbackRows} />}>
			<Match when={props.query.isPending}>
				<SkeletonList rows={props.fallbackRows} />
			</Match>
			<Match when={props.query.isError}>
				<ErrorState
					error={props.query.error}
					onRetry={() => props.query.refetch()}
				/>
			</Match>
			<Match when={data()}>
				{(d) => props.children(d as Accessor<NonNullable<T>>)}
			</Match>
			<Match when={props.notFound}>{props.notFound}</Match>
		</Switch>
	);
}
