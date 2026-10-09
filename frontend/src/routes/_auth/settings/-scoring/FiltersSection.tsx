import { createMemo } from "solid-js";
import { Field } from "@/components/Field";
import { MultiCombobox } from "@/components/MultiCombobox";
import { Input } from "@/components/ui/input";
import { uniqueCapitalised } from "@/lib/capitalise";
import { useCompanySearch } from "../../../../hooks/useCompanies";
import { useDebouncedTerm } from "../../../../hooks/useDebouncedTerm";
import { STANCE_TONE } from "./stance";
import type { ExclusionFilters } from "./useExclusionFilters";

export function FiltersSection(props: { filters: ExclusionFilters }) {
	const [term, searchCompanies] = useDebouncedTerm();
	const companiesQuery = useCompanySearch(term);
	const companyOptions = createMemo(() =>
		(companiesQuery.data ?? []).map((c) => ({ id: c.Name, label: c.Name })),
	);

	return (
		<>
			<p class="text-xs text-faint">
				Jobs matching any exclusion are dropped before scoring.
			</p>
			<MultiCombobox
				label="Excluded title keywords"
				hint={`Whole words only: "java" won't exclude "JavaScript".`}
				options={[]}
				value={props.filters.titleKeywords()}
				onChange={props.filters.setTitleKeywords}
				placeholder="Type a keyword, then Enter"
				chipClass={STANCE_TONE.avoid}
				creatable
			/>
			<MultiCombobox
				label="Excluded companies"
				options={companyOptions()}
				onSearch={searchCompanies}
				value={props.filters.companies()}
				onChange={(names) =>
					props.filters.setCompanies(uniqueCapitalised(names))
				}
				placeholder="Search or type a company…"
				chipClass={STANCE_TONE.avoid}
				creatable
			/>
			<MultiCombobox
				label="Excluded locations"
				options={[]}
				value={props.filters.locations()}
				onChange={(names) =>
					props.filters.setLocations(uniqueCapitalised(names))
				}
				placeholder="Type a location, then Enter"
				chipClass={STANCE_TONE.avoid}
				creatable
			/>
			<p class="text-xs text-faint">
				When a list below is set, jobs matching none of it are dropped. Remote
				or blank locations always pass the location list.
			</p>
			<MultiCombobox
				label="Required title keywords"
				options={[]}
				value={props.filters.requiredTitleKeywords()}
				onChange={props.filters.setRequiredTitleKeywords}
				placeholder="e.g. engineer, then Enter"
				creatable
			/>
			<MultiCombobox
				label="Required locations"
				options={[]}
				value={props.filters.requiredLocations()}
				onChange={(names) =>
					props.filters.setRequiredLocations(uniqueCapitalised(names))
				}
				placeholder="e.g. London, then Enter"
				creatable
			/>
			<Field label="Notify me at a score of" for="threshold" hint="Out of 100.">
				<Input
					id="threshold"
					type="number"
					min="0"
					max="100"
					value={props.filters.threshold()}
					onInput={(e) =>
						props.filters.setThreshold(Number(e.currentTarget.value))
					}
					class="w-24 font-mono tabular-nums"
				/>
			</Field>
			<Field
				label="Only score jobs posted in the last"
				for="max-job-age"
				hint="Days, up to 365. Older jobs are saved but never scored. 0 scores every job."
			>
				<Input
					id="max-job-age"
					type="number"
					min="0"
					max="365"
					value={props.filters.maxJobAgeDays()}
					onInput={(e) => {
						const days = e.currentTarget.valueAsNumber;
						if (!Number.isNaN(days)) props.filters.setMaxJobAgeDays(days);
					}}
					class="w-24 font-mono tabular-nums"
				/>
			</Field>
		</>
	);
}
