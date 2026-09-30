import { createSignal, onCleanup, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { UnresolvableBoardError } from "../../../../api/companies";
import { useAddCompany } from "../../../../hooks/useCompanies";
import { useResolveUrl } from "../../../../hooks/useSources";
import { ResolveError } from "../../../../lib/resolveError";
import type { SourceInfo } from "../../../../types/source";
import { SearchField } from "./parts";

const DEBOUNCE_MS = 300;

export function TrackCompanyBox(props: {
	sources: SourceInfo[];
	onTracked: (name: string) => void;
	onError: (message: string) => void;
}) {
	const addMutation = useAddCompany();
	const [raw, setRaw] = createSignal("");
	const [url, setUrl] = createSignal("");
	let timer: ReturnType<typeof setTimeout> | undefined;
	onCleanup(() => clearTimeout(timer));

	const input = (value: string) => {
		setRaw(value);
		clearTimeout(timer);
		timer = setTimeout(() => setUrl(value.trim()), DEBOUNCE_MS);
	};

	const resolved = useResolveUrl(url, () => /^https?:\/\/\S+$/i.test(url()));
	const board = () => {
		const r = resolved.data;
		if (!r || url() !== raw().trim()) return undefined;
		const info = props.sources.find((s) => s.name === r.source);
		return info?.role === "ats" ? info : undefined;
	};
	const unrecognised = () =>
		url() !== "" &&
		url() === raw().trim() &&
		(resolved.isSuccess || resolved.error instanceof ResolveError) &&
		board() === undefined;

	const track = async () => {
		const value = raw().trim();
		try {
			const company = await addMutation.mutateAsync({
				url: value,
				track: true,
			});
			setRaw("");
			setUrl("");
			props.onTracked(company.Name);
		} catch (err) {
			props.onError(
				err instanceof UnresolvableBoardError
					? "Couldn't detect an ATS board from that URL."
					: "Could not track the company. Please try again.",
			);
		}
	};

	return (
		<div class="flex flex-wrap items-center gap-2">
			<SearchField
				id="track-company-url"
				label="Company board URL"
				placeholder="Paste a Greenhouse, Lever or Ashby URL…"
				value={raw()}
				onInput={input}
				class="sm:max-w-96"
			/>
			<Show when={board()}>
				{(info) => (
					<Button size="sm" onClick={track} disabled={addMutation.isPending}>
						{addMutation.isPending
							? "Tracking…"
							: `Track company (${info().label})`}
					</Button>
				)}
			</Show>
			<Show when={unrecognised()}>
				<span class="text-xs text-muted">
					Not a recognised company board URL.
				</span>
			</Show>
		</div>
	);
}
