import { onCleanup, onMount } from "solid-js";

const BLOCKED_TARGETS =
	'input, textarea, select, [contenteditable]:not([contenteditable="false"]), [role="dialog"], [role="menu"], [data-no-hotkeys]';

function shouldIgnore(e: KeyboardEvent) {
	if (e.metaKey || e.ctrlKey || e.altKey) return true;
	return (
		e.target instanceof Element && e.target.closest(BLOCKED_TARGETS) !== null
	);
}

export function useShortcuts(
	shortcuts: Record<string, (e: KeyboardEvent) => void>,
) {
	const onKeyDown = (e: KeyboardEvent) => {
		const handler = Object.hasOwn(shortcuts, e.key)
			? shortcuts[e.key]
			: undefined;
		if (!handler || shouldIgnore(e)) return;
		e.preventDefault();
		handler(e);
	};
	onMount(() => document.addEventListener("keydown", onKeyDown));
	onCleanup(() => document.removeEventListener("keydown", onKeyDown));
}
