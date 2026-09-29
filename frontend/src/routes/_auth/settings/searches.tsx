import { createFileRoute } from "@tanstack/solid-router";
import { SearchesPrototype } from "./-searches-prototype/SearchesPrototype";
import { useProtoSearches } from "./-searches-prototype/useProtoSearches";

export const Route = createFileRoute("/_auth/settings/searches")({
	validateSearch: (search: Record<string, unknown>) => ({
		tab: search.tab === "ats" ? ("ats" as const) : undefined,
	}),
	component: SearchesPage,
});

function SearchesPage() {
	const search = Route.useSearch();
	const navigate = Route.useNavigate();
	const s = useProtoSearches();
	return (
		<SearchesPrototype
			s={s}
			tab={search().tab ?? "boards"}
			onTab={(t) =>
				navigate({
					search: { tab: t === "ats" ? "ats" : undefined },
					replace: true,
				})
			}
		/>
	);
}
