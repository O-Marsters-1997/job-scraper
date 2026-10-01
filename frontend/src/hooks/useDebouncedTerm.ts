import { type Accessor, createSignal, onCleanup } from "solid-js";

const SEARCH_DEBOUNCE_MS = 300;

export function useDebouncedTerm(): [
	Accessor<string>,
	(value: string) => void,
] {
	const [term, setTerm] = createSignal("");
	let timer: ReturnType<typeof setTimeout> | undefined;
	onCleanup(() => clearTimeout(timer));
	const search = (value: string) => {
		clearTimeout(timer);
		timer = setTimeout(() => setTerm(value.trim()), SEARCH_DEBOUNCE_MS);
	};
	return [term, search];
}
