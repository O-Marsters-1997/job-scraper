import { createEffect, createSignal, onCleanup, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { isDuplicateSearch } from "@/lib/searchTargets";
import { useResolveUrl } from "../../../../hooks/useSources";
import { ResolveError } from "../../../../lib/resolveError";
import type { ResolvedURL } from "../../../../types/source";
import type { SourceTarget } from "../../../../types/sourceTarget";

const DEBOUNCE_MS = 350;

export function PasteBox(props: {
	targets: SourceTarget[] | undefined;
	onSearch: (resolved: ResolvedURL) => void;
}) {
	const [text, setText] = createSignal("");
	const [url, setUrl] = createSignal("");
	let timer: ReturnType<typeof setTimeout> | undefined;
	onCleanup(() => clearTimeout(timer));

	const query = useResolveUrl(url);

	const duplicate = () => {
		const data = query.data;
		return (
			data?.kind === "search" && isDuplicateSearch(props.targets ?? [], data)
		);
	};

	createEffect(() => {
		const data = query.data;
		if (!data || data.kind !== "search" || duplicate()) return;
		props.onSearch(data);
		clearTimeout(timer);
		setText("");
		setUrl("");
	});

	const onInput = (value: string) => {
		setText(value);
		clearTimeout(timer);
		timer = setTimeout(() => setUrl(value.trim()), DEBOUNCE_MS);
	};

	const message = (): { tone: "info" | "error"; text: string } | null => {
		if (!url()) return null;
		if (query.isFetching) return { tone: "info", text: "Checking URL…" };
		if (query.error) {
			return {
				tone: "error",
				text:
					query.error instanceof ResolveError
						? query.error.message
						: "Could not check that URL. Please try again.",
			};
		}
		if (duplicate())
			return { tone: "error", text: "You already have this search." };
		if (query.data?.kind === "ats") {
			return {
				tone: "info",
				text: "That's a company board. Adding company boards from here is coming soon.",
			};
		}
		return null;
	};

	return (
		<div class="mb-3 flex flex-col gap-1.5">
			<Label for="paste-search-url" class="sr-only">
				Paste a job search URL
			</Label>
			<div class="relative">
				<Icon
					name="link"
					size={14}
					class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-faint"
				/>
				<Input
					id="paste-search-url"
					type="text"
					placeholder="Paste a LinkedIn, Indeed or WIS search URL"
					value={text()}
					onInput={(e) => onInput(e.currentTarget.value)}
					class="pl-9"
				/>
			</div>
			<Show when={message()}>
				{(m) => (
					<p
						role={m().tone === "error" ? "alert" : "status"}
						class={
							m().tone === "error"
								? "text-xs text-destructive-strong"
								: "text-xs text-muted"
						}
					>
						{m().text}
					</p>
				)}
			</Show>
		</div>
	);
}
