import type { JSX } from "solid-js";
import { Show } from "solid-js";
import { ErrorState } from "@/components/ErrorState";
import { SkeletonList } from "@/components/ui/skeleton";

// Encapsulates the four async states (loading / error / empty / success) so
// individual routes don't have to repeat the pattern or forget the error branch.
//
// Usage:
//   <QueryBoundary query={query} fallbackRows={4}>
//     {(data) => <MyContent data={data} />}
//   </QueryBoundary>
type Props<T> = {
	query: {
		isPending: boolean;
		isError: boolean;
		error: unknown;
		refetch: () => unknown;
		data: T | undefined;
	};
	fallbackRows?: number;
	children: (data: T) => JSX.Element;
};

export function QueryBoundary<T>(props: Props<T>): JSX.Element {
	// Use Show's accessor form (when={data}) so children are lazy —
	// they mount only once data is defined, not while loading/erroring.
	return (
		<Show
			when={props.query.data}
			fallback={
				<Show
					when={props.query.isError}
					fallback={<SkeletonList rows={props.fallbackRows} />}
				>
					<ErrorState
						error={props.query.error}
						onRetry={() => props.query.refetch()}
					/>
				</Show>
			}
		>
			{(data) => props.children(data() as T)}
		</Show>
	);
}
