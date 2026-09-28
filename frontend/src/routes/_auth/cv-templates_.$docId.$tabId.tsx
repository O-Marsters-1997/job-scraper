import { createFileRoute, Link } from "@tanstack/solid-router";
import {
	createResource,
	onCleanup,
	type ResourceFetcherInfo,
	Show,
} from "solid-js";
import { Icon } from "@/components/Icon";
import { fetchCVPdf } from "../../api/cvTemplates";

export const Route = createFileRoute("/_auth/cv-templates_/$docId/$tabId")({
	component: CVDetailPage,
});

// Google Docs tabId values already include the "t." prefix (e.g. "t.0").
// Prepend it only if absent to avoid producing "t.t.0".
function tabParam(tabId: string): string {
	return tabId.startsWith("t.") ? tabId : `t.${tabId}`;
}

type CVPdfSource = { docId: string; tabId: string };

// Revokes the previous object URL here too, not just in onCleanup, so a
// doc/tab switch doesn't leak it before the component unmounts.
async function loadCVPdfURL(
	source: CVPdfSource,
	{ value }: ResourceFetcherInfo<string>,
): Promise<string> {
	if (value) URL.revokeObjectURL(value);
	const data = await fetchCVPdf(source.docId, source.tabId);
	return URL.createObjectURL(new Blob([data], { type: "application/pdf" }));
}

function CVDetailPage() {
	const params = Route.useParams();

	const docsUrl = () =>
		`https://docs.google.com/document/d/${params().docId}/edit?tab=${tabParam(params().tabId)}`;

	const [pdfURL] = createResource(
		(): CVPdfSource => ({ docId: params().docId, tabId: params().tabId }),
		loadCVPdfURL,
	);

	onCleanup(() => {
		if (pdfURL.latest) URL.revokeObjectURL(pdfURL.latest);
	});

	return (
		<div class="flex min-h-screen flex-col bg-background">
			<div class="sticky top-0 z-10 flex h-12 items-center justify-between border-b border-border bg-surface px-7">
				<Link
					to="/cv-templates"
					class="flex items-center gap-1.5 text-sm text-muted transition-colors hover:text-foreground"
				>
					<Icon name="chevronLeft" size={14} />
					CVs
				</Link>

				<div class="flex items-center gap-2">
					{/* iOS Safari renders only page 1 of a PDF in an iframe, so this link
					gives it (and anyone else) a way to see every page. */}
					<Show when={pdfURL()}>
						{(url) => (
							<a
								href={url()}
								target="_blank"
								rel="noopener noreferrer"
								class="inline-flex h-8 items-center gap-1.5 rounded-md border border-border bg-surface px-3 text-xs font-medium text-foreground transition-colors hover:border-border-strong hover:bg-surface-muted"
							>
								<Icon name="externalLink" size={12} />
								Open PDF
							</a>
						)}
					</Show>
					<a
						href={docsUrl()}
						target="_blank"
						rel="noopener noreferrer"
						class="inline-flex h-8 items-center gap-1.5 rounded-md border border-border bg-surface px-3 text-xs font-medium text-foreground transition-colors hover:border-border-strong hover:bg-surface-muted"
					>
						<Icon name="externalLink" size={12} />
						Open in Google Docs
					</a>
				</div>
			</div>

			<div class="flex flex-1 flex-col items-center px-7 py-8">
				<Show when={pdfURL.loading}>
					<div class="h-[1120px] w-full max-w-3xl animate-pulse rounded-xl bg-surface-muted" />
				</Show>

				<Show when={pdfURL.error}>
					{(err) => (
						<div class="w-full max-w-3xl rounded-xl border border-destructive/30 bg-destructive-subtle p-6">
							<p class="mb-1 text-sm font-semibold text-destructive-strong">
								Failed to load PDF
							</p>
							<p class="text-sm text-muted">
								{err() instanceof Error ? err().message : "Failed to load PDF"}
							</p>
						</div>
					)}
				</Show>

				<Show when={!pdfURL.loading && !pdfURL.error && pdfURL()}>
					{(url) => (
						<iframe
							src={url()}
							title="CV PDF"
							class="h-[1120px] w-full max-w-3xl rounded-xl border border-border bg-surface"
						/>
					)}
				</Show>
			</div>
		</div>
	);
}
