import {
	createResource,
	onCleanup,
	type Resource,
	type ResourceFetcherInfo,
} from "solid-js";

export function usePdfUrl<S>(
	source: () => S | undefined,
	fetchPdf: (source: S) => Promise<ArrayBuffer>,
): Resource<string> {
	const [pdfURL] = createResource<string, NonNullable<S>>(
		() => source() ?? false,
		async (s: NonNullable<S>, { value }: ResourceFetcherInfo<string>) => {
			if (value) URL.revokeObjectURL(value);
			const data = await fetchPdf(s);
			return URL.createObjectURL(new Blob([data], { type: "application/pdf" }));
		},
	);
	onCleanup(() => {
		if (pdfURL.latest) URL.revokeObjectURL(pdfURL.latest);
	});
	return pdfURL;
}
