import { onCleanup, onMount } from "solid-js";
import { Icon } from "@/components/Icon";
import { PdfPreview } from "@/components/PdfPreview";
import { Button } from "@/components/ui/button";
import { fetchDraftPdf } from "../../api/tailoring";
import { usePdfUrl } from "../../hooks/usePdfUrl";

export function PrintPreview(props: { draftId: string; onClose: () => void }) {
	const url = usePdfUrl(
		() => ({ id: props.draftId }),
		(s) => fetchDraftPdf(s.id),
	);
	let closeBtn: HTMLButtonElement | undefined;
	onMount(() => {
		const opener = document.activeElement;
		closeBtn?.focus();
		const onKey = (e: KeyboardEvent) => e.key === "Escape" && props.onClose();
		window.addEventListener("keydown", onKey);
		onCleanup(() => {
			window.removeEventListener("keydown", onKey);
			if (opener instanceof HTMLElement) opener.focus();
		});
	});
	return (
		<div class="fixed inset-0 z-[55] flex justify-end bg-black/30 backdrop-blur-[2px]">
			<button
				type="button"
				tabIndex={-1}
				aria-hidden="true"
				class="flex-1 cursor-default"
				onClick={() => props.onClose()}
			/>
			<div
				role="dialog"
				aria-modal="true"
				aria-label="Print preview"
				class="scroll-slim flex h-full w-full max-w-[52rem] flex-col overflow-y-auto bg-background shadow-2xl"
			>
				<div class="sticky top-0 z-10 flex h-14 items-center justify-between border-b border-border bg-surface px-5">
					<div>
						<p class="text-sm font-semibold text-foreground">Print preview</p>
						<p class="text-xs text-faint">
							Google's own export of the Draft, the final word on page breaks.
						</p>
					</div>
					<Button
						ref={closeBtn}
						variant="ghost"
						size="icon"
						aria-label="Close preview"
						onClick={() => props.onClose()}
					>
						<Icon name="x" size={16} />
					</Button>
				</div>
				<div class="p-6">
					<PdfPreview url={url} title="Draft CV PDF" />
				</div>
			</div>
		</div>
	);
}
