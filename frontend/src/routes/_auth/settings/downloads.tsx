import { createFileRoute } from "@tanstack/solid-router";
import type { Accessor } from "solid-js";
import { createResource, createSignal, Show, untrack } from "solid-js";
import { Field } from "@/components/Field";
import { FormFeedback } from "@/components/FormFeedback";
import { QueryBoundary } from "@/components/QueryBoundary";
import { SettingsActions } from "@/components/SettingsLayout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cvFilename } from "@/lib/cvFilename";
import {
	chooseDownloadFolder,
	clearDownloadFolder,
	folderPickerSupported,
	getDownloadFolder,
} from "@/lib/downloadFolder";
import {
	useProfile,
	useUpdateCvFilenamePattern,
} from "../../../hooks/useProfile";
import type { Profile } from "../../../types/profile";

export const Route = createFileRoute("/_auth/settings/downloads")({
	component: DownloadsPage,
});

function DownloadsPage() {
	const query = useProfile();
	return (
		<QueryBoundary query={query} fallbackRows={3}>
			{(data) => (
				<>
					<FolderSection />
					<PatternForm data={data} />
				</>
			)}
		</QueryBoundary>
	);
}

function FolderSection() {
	const [folder, { mutate }] = createResource(getDownloadFolder);
	const [error, setError] = createSignal<string | null>(null);

	const choose = async () => {
		setError(null);
		try {
			const handle = await chooseDownloadFolder();
			if (handle) mutate(handle);
		} catch {
			setError("Could not use that folder. Try another.");
		}
	};
	const clear = async () => {
		setError(null);
		try {
			await clearDownloadFolder();
			mutate(undefined);
		} catch {
			setError("Could not clear the folder. Try again.");
		}
	};

	return (
		<div>
			<p class="text-sm font-medium text-foreground">Download folder</p>
			<Show
				when={folderPickerSupported()}
				fallback={
					<p class="mt-2 text-sm text-muted">
						This browser cannot remember a folder, so CVs go to its normal
						downloads. Use Chrome, Edge or another Chromium browser to pick one.
					</p>
				}
			>
				<FormFeedback error={error()} />
				<p class="mt-0.5 text-xs text-faint">
					CVs are saved here without a dialog. Remembered in this browser only.
				</p>
				<div class="mt-2 flex flex-wrap items-center gap-3">
					<p class="text-sm text-muted">
						{folder.loading ? "" : (folder()?.name ?? "No folder chosen")}
					</p>
					<Button
						variant="outline"
						size="sm"
						disabled={folder.loading}
						onClick={() => void choose()}
					>
						{folder() ? "Change folder" : "Choose folder"}
					</Button>
					<Show when={folder()}>
						<Button variant="ghost" size="sm" onClick={() => void clear()}>
							Clear
						</Button>
					</Show>
				</div>
			</Show>
		</div>
	);
}

function PatternForm(props: { data: Accessor<Profile> }) {
	const mutation = useUpdateCvFilenamePattern();
	const [pattern, setPattern] = createSignal(
		untrack(() => props.data().cvFilenamePattern),
	);
	const [saved, setSaved] = createSignal(false);
	const [saveError, setSaveError] = createSignal<string | null>(null);

	const example = () =>
		cvFilename(pattern(), {
			name: props.data().username,
			company: "acme-corp",
			role: "Senior Engineer",
			date: new Date().toLocaleDateString("sv-SE"),
		});

	const handleSave = async () => {
		setSaved(false);
		setSaveError(null);
		try {
			await mutation.mutateAsync(pattern());
			setSaved(true);
			setTimeout(() => setSaved(false), 3000);
		} catch {
			setSaveError("Failed to save. Please try again.");
		}
	};

	return (
		<>
			<SettingsActions>
				<Button size="sm" onClick={handleSave} loading={mutation.isPending}>
					Save
				</Button>
			</SettingsActions>
			<FormFeedback success={saved()} error={saveError()} />
			<Field
				label="Filename pattern"
				for="cv-filename"
				hint="Tokens: {name}, {company}, {role}, {date}. .pdf is added for you."
			>
				<Input
					id="cv-filename"
					value={pattern()}
					maxLength={120}
					aria-describedby="cv-filename-example"
					onInput={(e) => setPattern(e.currentTarget.value)}
					class="max-w-sm"
				/>
				<p id="cv-filename-example" class="mt-2 text-xs text-muted">
					Example: <span class="font-mono">{example()}</span>
				</p>
			</Field>
		</>
	);
}
