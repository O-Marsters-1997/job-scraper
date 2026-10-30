export function isTyping(el: EventTarget | null) {
	return (
		el instanceof HTMLElement &&
		(el.isContentEditable ||
			["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName))
	);
}
