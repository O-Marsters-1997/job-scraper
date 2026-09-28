import { createFileRoute, Link } from "@tanstack/solid-router";
import type { PDFDocumentLoadingTask, PDFPageProxy } from "pdfjs-dist";
import * as pdfjsLib from "pdfjs-dist";
import {
	createResource,
	For,
	onCleanup,
	type ResourceFetcherInfo,
	Show,
	untrack,
} from "solid-js";
import { Icon } from "@/components/Icon";
import { fetchCVPdf } from "../../api/cvTemplates";

pdfjsLib.GlobalWorkerOptions.workerSrc = new URL(
	"pdfjs-dist/build/pdf.worker.min.mjs",
	import.meta.url,
).href;

export const Route = createFileRoute("/_auth/cv-templates_/$docId/$tabId")({
	component: CVDetailPage,
});

// Google Docs tabId values already include the "t." prefix (e.g. "t.0").
// Prepend it only if absent to avoid producing "t.t.0".
function tabParam(tabId: string): string {
	return tabId.startsWith("t.") ? tabId : `t.${tabId}`;
}

type CVPdf = { loadingTask: PDFDocumentLoadingTask; pages: PDFPageProxy[] };
type CVPdfSource = { docId: string; tabId: string };

// Guards against a fetch that resolves after a newer one has already started:
// without this, that PDF's loadingTask never becomes the resource's `value`
// (solid drops stale resolutions) and so never gets destroy()ed.
let cvPdfRequestSeq = 0;

async function loadCVPdf(
	source: CVPdfSource,
	{ value }: ResourceFetcherInfo<CVPdf>,
): Promise<CVPdf> {
	const seq = ++cvPdfRequestSeq;
	await value?.loadingTask.destroy();
	const data = await fetchCVPdf(source.docId, source.tabId);
	const loadingTask = pdfjsLib.getDocument({ data });
	const pdf = await loadingTask.promise;
	const pages = await Promise.all(
		Array.from({ length: pdf.numPages }, (_, i) => pdf.getPage(i + 1)),
	);
	if (seq !== cvPdfRequestSeq) {
		await loadingTask.destroy();
		throw new Error("superseded by a newer request");
	}
	return { loadingTask, pages };
}

function CVDetailPage() {
	const params = Route.useParams();

	const docsUrl = () =>
		`https://docs.google.com/document/d/${params().docId}/edit?tab=${tabParam(params().tabId)}`;

	const [pdfResource] = createResource(
		(): CVPdfSource => ({ docId: params().docId, tabId: params().tabId }),
		loadCVPdf,
	);

	onCleanup(() => {
		pdfResource.latest?.loadingTask.destroy();
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

			<div class="flex flex-1 flex-col items-center px-7 py-8">
				<Show when={pdfResource.loading}>
					<div class="flex w-full max-w-3xl flex-col gap-4">
						<For each={[1, 2, 3]}>
							{() => (
								<div class="h-[1120px] w-full animate-pulse rounded-xl bg-surface-muted" />
							)}
						</For>
					</div>
				</Show>

				<Show when={pdfResource.error}>
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

				<Show
					when={!pdfResource.loading && !pdfResource.error && pdfResource()}
				>
					{(res) => (
						<div class="flex w-full max-w-3xl flex-col gap-4">
							<For each={res().pages}>
								{(page) => <PDFCanvas page={page} />}
							</For>
						</div>
					)}
				</Show>
			</div>
		</div>
	);
}

function PDFCanvas(props: { page: PDFPageProxy }) {
	let canvasRef: HTMLCanvasElement | undefined;

	const viewport = untrack(() => props.page.getViewport({ scale: 1.5 }));

	const render = () => {
		if (!canvasRef) return;
		canvasRef.width = viewport.width;
		canvasRef.height = viewport.height;
		props.page.render({ canvas: canvasRef, viewport });
	};

	return (
		<div class="overflow-hidden rounded-xl border border-border bg-surface shadow-none">
			<canvas
				ref={(el) => {
					canvasRef = el;
					render();
				}}
				width={viewport.width}
				height={viewport.height}
				class="block w-full"
			/>
		</div>
	);
}
