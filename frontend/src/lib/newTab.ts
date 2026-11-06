export async function openInNewTab(resolveUrl: () => Promise<string>) {
	const tab = window.open("", "_blank");
	try {
		const url = await resolveUrl();
		if (tab) tab.location.href = url;
		else window.location.assign(url);
	} catch (err) {
		tab?.close();
		throw err;
	}
}
