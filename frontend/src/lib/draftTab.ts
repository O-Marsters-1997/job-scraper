export const draftPath = (id: string) => `/tailoring/drafts/${id}`;

export async function openDraftInNewTab(
	create: () => Promise<{ id: string }>,
): Promise<void> {
	const tab = window.open("", "_blank");
	try {
		const { id } = await create();
		if (tab) tab.location.href = draftPath(id);
		else window.location.assign(draftPath(id));
	} catch (err) {
		tab?.close();
		throw err;
	}
}
