import { createFileRoute, Link } from "@tanstack/solid-router";
import * as pdfjsLib from "pdfjs-dist";
import type { PDFPageProxy } from "pdfjs-dist";
import { createResource, createSignal, For, onCleanup, Show } from "solid-js";

pdfjsLib.GlobalWorkerOptions.workerSrc = new URL(
	"pdfjs-dist/build/pdf.worker.min.mjs",
	import.meta.url,
).href;

export const Route = createFileRoute("/_auth/cv-templates_/$docId/$tabId")({
	component: CVDetailPage,
});

function CVDetailPage() {
	const { docId, tabId } = Route.useParams();

	// TODO: when spike #37 resolves per-tab isolation, append ?tab=t.{tabId}
	// to this URL so the backend can export only the relevant tab.
	const pdfUrl = () => `/cv-templates/${docId}/${tabId}/pdf`;
	const docsUrl = () =>
		`https://docs.google.com/document/d/${docId()}/edit?tab=t.${tabId()}`;

	const [blobUrl, setBlobUrl] = createSignal<string | null>(null);
	const [pages, setPages] = createSignal<PDFPageProxy[]>([]);
	const [error, setError] = createSignal<string | null>(null);

	const [pdfResource] = createResource(pdfUrl, async (url) => {
		setBlobUrl(null);
		setPages([]);
		setError(null);

		let objectUrl: string | null = null;
		try {
			const res = await fetch(url, { credentials: "include" });
			if (!res.ok) {
				throw new Error(`Server returned ${res.status}`);
			}
			const blob = await res.blob();
			objectUrl = URL.createObjectURL(blob);
			setBlobUrl(objectUrl);

			const pdf = await pdfjsLib.getDocument(objectUrl).promise;
			const loadedPages: PDFPageProxy[] = [];
			for (let i = 1; i <= pdf.numPages; i++) {
				loadedPages.push(await pdf.getPage(i));
			}
			setPages(loadedPages);
			return pdf;
		} catch (err) {
			setError(err instanceof Error ? err.message : "Failed to load PDF");
			if (objectUrl) URL.revokeObjectURL(objectUrl);
			return null;
		}
	});

	onCleanup(() => {
		const url = blobUrl();
		if (url) URL.revokeObjectURL(url);
	});

	return (
		<div class="flex min-h-screen flex-col bg-background">
			<div class="sticky top-0 z-10 flex h-12 items-center justify-between border-b border-border bg-surface px-7">
				<Link
					to="/cv-templates"
					class="flex items-center gap-1.5 text-sm text-muted transition-colors hover:text-foreground"
				>
					<svg
						aria-hidden="true"
						width="14"
						height="14"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<polyline points="15 18 9 12 15 6" />
					</svg>
					CV Templates
				</Link>

				<a
					href={docsUrl()}
					target="_blank"
					rel="noopener noreferrer"
					class="inline-flex h-8 items-center gap-1.5 rounded-md border border-border bg-surface px-3 text-xs font-medium text-foreground transition-colors hover:border-border-strong hover:bg-surface-muted"
				>
					<svg
						aria-hidden="true"
						width="12"
						height="12"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
						<polyline points="15 3 21 3 21 9" />
						<line x1="10" y1="14" x2="21" y2="3" />
					</svg>
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

				<Show when={error()}>
					{(msg) => (
						<div class="w-full max-w-3xl rounded-xl border border-destructive/30 bg-destructive-subtle p-6">
							<p class="mb-1 text-sm font-semibold text-destructive-strong">
								Failed to load PDF
							</p>
							<p class="text-sm text-muted">{msg()}</p>
						</div>
					)}
				</Show>

				<Show when={!pdfResource.loading && pages().length > 0}>
					<div class="flex w-full max-w-3xl flex-col gap-4">
						<For each={pages()}>
							{(page) => <PDFCanvas page={page} />}
						</For>
					</div>
				</Show>
			</div>
		</div>
	);
}

function PDFCanvas(props: { page: PDFPageProxy }) {
	let canvasRef: HTMLCanvasElement | undefined;

	const viewport = props.page.getViewport({ scale: 1.5 });

	const render = () => {
		if (!canvasRef) return;
		const ctx = canvasRef.getContext("2d");
		if (!ctx) return;
		canvasRef.width = viewport.width;
		canvasRef.height = viewport.height;
		props.page.render({ canvasContext: ctx, viewport });
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
