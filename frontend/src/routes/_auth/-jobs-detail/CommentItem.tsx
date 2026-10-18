import { createSignal, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
	useDeleteJobComment,
	useUpdateJobComment,
} from "@/hooks/useJobComments";
import { formatDate } from "@/lib/datetime";
import { JOB_COMMENT_MAX_LENGTH, type JobComment } from "@/types/jobComment";

export function CommentItem(props: {
	comment: JobComment;
	jobId: string;
	onRemoved?: () => void;
}) {
	const update = useUpdateJobComment(() => props.jobId);
	const remove = useDeleteJobComment(() => props.jobId);
	const [editing, setEditing] = createSignal(false);
	const [draft, setDraft] = createSignal("");

	const startEdit = () => {
		setDraft(props.comment.Body);
		setEditing(true);
	};
	const save = (e: SubmitEvent) => {
		e.preventDefault();
		update.mutate(
			{ id: props.comment.ID, body: draft() },
			{ onSuccess: () => setEditing(false) },
		);
	};
	const when = () => formatDate(props.comment.CreatedAt);

	return (
		<div class="flex flex-col gap-0.5">
			<span class="font-mono text-xs tabular-nums text-muted">
				{when()}
				<Show when={props.comment.UpdatedAt !== props.comment.CreatedAt}>
					{" "}
					(edited)
				</Show>
			</span>
			<Show
				when={editing()}
				fallback={
					<>
						<p class="whitespace-pre-wrap break-words text-sm text-foreground">
							{props.comment.Body}
						</p>
						<FormFeedback
							error={remove.isError ? "Could not delete the comment." : null}
						/>
						<div class="flex gap-2">
							<Button
								type="button"
								size="sm"
								variant="ghost"
								aria-label={`Edit comment from ${when()}`}
								onClick={startEdit}
							>
								Edit
							</Button>
							<Button
								type="button"
								size="sm"
								variant="ghost"
								aria-label={`Delete comment from ${when()}`}
								disabled={remove.isPending}
								onClick={() =>
									remove.mutate(props.comment.ID, {
										onSuccess: () => props.onRemoved?.(),
									})
								}
							>
								Delete
							</Button>
						</div>
					</>
				}
			>
				<form onSubmit={save} class="flex flex-col gap-2">
					<FormFeedback
						error={update.isError ? "Could not save the comment." : null}
					/>
					<Label for={`job-comment-edit-${props.comment.ID}`}>
						Edit comment
					</Label>
					<Textarea
						id={`job-comment-edit-${props.comment.ID}`}
						rows={3}
						maxLength={JOB_COMMENT_MAX_LENGTH}
						value={draft()}
						onInput={(e) => setDraft(e.currentTarget.value)}
					/>
					<div class="flex gap-2">
						<Button
							type="submit"
							size="sm"
							disabled={update.isPending || draft().length === 0}
						>
							Save
						</Button>
						<Button
							type="button"
							size="sm"
							variant="ghost"
							onClick={() => setEditing(false)}
						>
							Cancel
						</Button>
					</div>
				</form>
			</Show>
		</div>
	);
}
