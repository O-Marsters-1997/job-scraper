import { createSignal, For, onCleanup, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
	useAddCompanyBoard,
	useCompanyBoards,
} from "../../../hooks/useCompanies";
import type { CompanyBoard } from "../../../types/company";

const BOARD_CHECK_POLL_MS = 2000;
const BOARD_CHECK_TIMEOUT_MS = 30_000;

function boardURLFor(board: CompanyBoard): string {
	switch (board.Source) {
		case "greenhouse":
			return `https://boards.greenhouse.io/${board.BoardToken}`;
		case "lever":
			return `https://jobs.lever.co/${board.BoardToken}`;
		case "ashby":
			return `https://jobs.ashbyhq.com/${board.BoardToken}`;
		case "workable":
			return `https://apply.workable.com/${board.BoardToken}`;
		case "recruitee":
			return `https://${board.BoardToken}.recruitee.com`;
		case "personio":
			return `https://${board.BoardToken}.jobs.personio.de`;
		case "pinpoint":
			return `https://${board.BoardToken}.pinpointhq.com`;
		default:
			return "";
	}
}

export function CompanyBoardsCard(props: { companyId: string }) {
	const [checkingBoardID, setCheckingBoardID] = createSignal<string>();
	const boardsQuery = useCompanyBoards(
		() => props.companyId,
		(boards) => {
			const id = checkingBoardID();
			if (!id) return false;
			const verified = boards?.find((b) => b.ID === id)?.Status === "verified";
			return verified ? false : BOARD_CHECK_POLL_MS;
		},
	);
	const addBoardMutation = useAddCompanyBoard();
	const [boardURL, setBoardURL] = createSignal("");
	const [boardMessage, setBoardMessage] = createSignal("");
	let checkTimeout: ReturnType<typeof setTimeout> | undefined;
	onCleanup(() => clearTimeout(checkTimeout));
	const checkedBoardVerified = () =>
		boardsQuery.data?.find((b) => b.ID === checkingBoardID())?.Status ===
		"verified";
	const boardStatusMessage = () =>
		checkedBoardVerified() ? "Board verified." : boardMessage();
	const saveBoard = async (url: string) => {
		setBoardMessage("");
		setCheckingBoardID(undefined);
		clearTimeout(checkTimeout);
		try {
			const board = await addBoardMutation.mutateAsync({
				id: props.companyId,
				url,
				confirm: true,
			});
			setBoardURL("");
			if (board.Status === "verified") {
				setBoardMessage("Board verified.");
				return;
			}
			setBoardMessage("Checking the board…");
			setCheckingBoardID(board.ID);
			checkTimeout = setTimeout(() => {
				if (checkedBoardVerified()) return;
				setCheckingBoardID(undefined);
				setBoardMessage(
					"Verification failed; retry when the board is available.",
				);
			}, BOARD_CHECK_TIMEOUT_MS);
		} catch (error) {
			setBoardMessage(
				error instanceof Error ? error.message : "Could not add board.",
			);
		}
	};

	return (
		<Card>
			<CardHeader class="pb-2">
				<CardTitle>Boards</CardTitle>
			</CardHeader>
			<CardContent class="gap-3">
				<Show when={boardsQuery.isError}>
					<p class="text-sm text-destructive-strong">Could not load boards.</p>
				</Show>
				<Show when={boardsQuery.isPending}>
					<p class="text-sm text-muted">Loading boards…</p>
				</Show>
				<Show when={!boardsQuery.isPending && !boardsQuery.isError}>
					<Show
						when={(boardsQuery.data ?? []).length > 0}
						fallback={<p class="text-sm text-faint">No boards linked yet.</p>}
					>
						<For each={boardsQuery.data ?? []}>
							{(board) => (
								<div class="flex items-start justify-between gap-2 border-b border-border pb-2 text-xs">
									<div class="min-w-0">
										<p class="font-medium text-foreground">{board.Source}</p>
										<p class="break-all font-mono text-faint">
											{board.BoardToken}
										</p>
										<Show when={board.LastCompletedAt}>
											{(checked) => (
												<p class="text-faint">
													Last checked {new Date(checked()).toLocaleString()}
												</p>
											)}
										</Show>
									</div>
									<span class="capitalize text-muted">{board.Status}</span>
								</div>
							)}
						</For>
					</Show>
				</Show>
				<form
					onSubmit={(event) => {
						event.preventDefault();
						void saveBoard(boardURL().trim());
					}}
					class="flex flex-col gap-2"
				>
					<label
						for="company-board-url"
						class="text-xs font-medium text-foreground"
					>
						ATS board URL
					</label>
					<Input
						id="company-board-url"
						type="url"
						required
						value={boardURL()}
						onInput={(event) => setBoardURL(event.currentTarget.value)}
						placeholder="https://boards.greenhouse.io/acme"
					/>
					<Button type="submit" size="sm" disabled={addBoardMutation.isPending}>
						Add and verify board
					</Button>
					<Show when={boardStatusMessage()}>
						<output class="text-xs text-muted">{boardStatusMessage()}</output>
					</Show>
				</form>
				<For
					each={(boardsQuery.data ?? []).filter(
						(board) => board.Status === "candidate",
					)}
				>
					{(board) => (
						<Button
							type="button"
							size="sm"
							variant="outline"
							disabled={addBoardMutation.isPending}
							onClick={() => saveBoard(boardURLFor(board))}
						>
							Retry {board.BoardToken}
						</Button>
					)}
				</For>
			</CardContent>
		</Card>
	);
}
