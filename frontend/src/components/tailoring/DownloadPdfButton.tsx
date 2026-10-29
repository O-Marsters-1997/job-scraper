import { Show } from "solid-js";
import { ApiError } from "@/api/apiError";
import { Button, type ButtonProps } from "@/components/ui/button";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { useJob } from "@/hooks/useJobs";
import { useProfile } from "@/hooks/useProfile";
import { cvFilename, DEFAULT_CV_FILENAME_PATTERN } from "@/lib/cvFilename";
import { savePdf, writableDownloadFolder } from "@/lib/downloadFolder";
import { loadedData } from "@/lib/queryData";
import { fetchDraftPdf } from "../../api/tailoring";

function failureMessage(err: unknown) {
	return err instanceof ApiError && err.status === 502
		? "Google could not export the PDF. Try again in a moment."
		: "Could not download the PDF. Try again.";
}

export function DownloadPdfButton(props: {
	draftId: string;
	jobId: string;
	flush?: () => Promise<boolean>;
	variant?: ButtonProps["variant"];
}) {
	const job = useJob(() => props.jobId);
	const profile = useProfile();
	const download = useFormSubmit(async () => {
		const folder = await writableDownloadFolder();
		if (props.flush && !(await props.flush())) return;
		const p = loadedData(profile);
		const name = cvFilename(
			p?.cvFilenamePattern ?? DEFAULT_CV_FILENAME_PATTERN,
			{
				name: p?.username ?? "",
				company: loadedData(job)?.CompanySlug ?? "draft",
				role: loadedData(job)?.Title ?? "",
				date: new Date().toLocaleDateString("sv-SE"),
			},
		);
		await savePdf(await fetchDraftPdf(props.draftId), name, folder);
	}, failureMessage);

	return (
		<>
			<Button
				variant={props.variant ?? "ghost"}
				size="sm"
				loading={download.pending()}
				onClick={() => void download.submit()}
			>
				Download PDF
			</Button>
			<Show when={download.error()}>
				<span role="alert" class="text-xs text-destructive-strong">
					{download.error()}
				</span>
			</Show>
		</>
	);
}
