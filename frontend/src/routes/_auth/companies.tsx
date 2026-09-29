import { createFileRoute, Link } from "@tanstack/solid-router";
import { createMemo, createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Switch,
	SwitchControl,
	SwitchLabel,
	SwitchThumb,
} from "@/components/ui/switch";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { formatDate } from "@/lib/datetime";
import { UnresolvableBoardError } from "../../api/companies";
import {
	companiesQueryOptions,
	useAddCompany,
	useCompanies,
	useSetCompanyTracking,
} from "../../hooks/useCompanies";
import { queryClient } from "../../lib/queryClient";
import type { Company } from "../../types/company";

export const Route = createFileRoute("/_auth/companies")({
	loader: () => queryClient.ensureQueryData(companiesQueryOptions),
	component: CompaniesPage,
});

function CompaniesPage() {
	const query = useCompanies();
	const addMutation = useAddCompany();
	const trackMutation = useSetCompanyTracking();

	const [search, setSearch] = createSignal("");
	const [showAdd, setShowAdd] = createSignal(false);
	const [newUrl, setNewUrl] = createSignal("");
	const [addedCompany, setAddedCompany] = createSignal(false);

	const filtered = createMemo(() => {
		const q = search().trim().toLowerCase();
		const all = query.data ?? [];
		if (!q) return all;
		return all.filter((c) => c.Name.toLowerCase().includes(q));
	});

	const handleToggle = (c: Company) => {
		trackMutation.mutate({ id: c.ID, enabled: !c.Tracked });
	};

	const addForm = useFormSubmit(
		async () => {
			const url = newUrl().trim();
			if (!url) return;
			setAddedCompany(false);
			await addMutation.mutateAsync({ url, track: true });
			resetForm();
			setAddedCompany(true);
			setShowAdd(false);
		},
		(err) =>
			err instanceof UnresolvableBoardError
				? "Couldn't detect an ATS board from that URL. Try the direct board link, e.g. https://boards.greenhouse.io/acmecorp."
				: "Failed to add company. Please try again.",
	);

	function resetForm() {
		setNewUrl("");
		addForm.setError(null);
	}

	return (
		<div class="px-7 py-6">
			<PageHeading
				title="Companies"
				subtitle="Every company we've encountered. Track a company to follow its current and future boards."
			>
				<Button
					onClick={() => {
						setAddedCompany(false);
						addForm.setError(null);
						setShowAdd(true);
					}}
				>
					Add company
				</Button>
			</PageHeading>

			<FormFeedback
				success={
					addedCompany()
						? "Company added. Open its detail page to verify the candidate board."
						: false
				}
			/>

			<div class="mb-4 max-w-xs">
				<Label for="companies-search" class="sr-only">
					Search companies
				</Label>
				<Input
					id="companies-search"
					placeholder="Search companies…"
					value={search()}
					onInput={(e) => setSearch(e.currentTarget.value)}
				/>
			</div>

			<QueryBoundary query={query} fallbackRows={6}>
				{() => (
					<Card class="overflow-hidden">
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>Name</TableHead>
									<TableHead>ATS</TableHead>
									<TableHead>Jobs</TableHead>
									<TableHead>First seen</TableHead>
									<TableHead class="w-16">Tracked</TableHead>
								</TableRow>
							</TableHeader>
							<TableBody>
								<For each={filtered()}>
									{(c) => (
										<TableRow>
											<TableCell>
												<Link
													to="/companies/$id"
													params={{ id: c.ID }}
													class="font-medium text-foreground hover:underline"
												>
													{c.Name}
												</Link>
											</TableCell>
											<TableCell>
												<Show
													when={c.ATSSource}
													fallback={
														<span class="text-xs text-faint">discovery</span>
													}
												>
													{(source) => (
														<Badge variant="source">{source()}</Badge>
													)}
												</Show>
											</TableCell>
											<TableCell class="font-mono text-xs tabular-nums text-muted">
												{c.JobCount}
											</TableCell>
											<TableCell class="font-mono text-xs tabular-nums text-faint">
												{formatDate(c.FirstSeenAt)}
											</TableCell>
											<TableCell>
												<Switch
													checked={c.Tracked}
													onChange={() => handleToggle(c)}
													disabled={trackMutation.isPending}
												>
													<SwitchLabel class="sr-only">
														Track {c.Name}
													</SwitchLabel>
													<SwitchControl>
														<SwitchThumb />
													</SwitchControl>
												</Switch>
											</TableCell>
										</TableRow>
									)}
								</For>
							</TableBody>
						</Table>
					</Card>
				)}
			</QueryBoundary>

			<Dialog open={showAdd()} onOpenChange={setShowAdd}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Add company</DialogTitle>
					</DialogHeader>
					<form onSubmit={addForm.submit} class="flex flex-col gap-3">
						<div class="flex flex-col gap-2">
							<label
								for="add-company-url"
								class="text-xs font-medium text-foreground"
							>
								ATS board URL
							</label>
							<Input
								id="add-company-url"
								placeholder="e.g. https://boards.greenhouse.io/acmecorp"
								value={newUrl()}
								onInput={(e) => setNewUrl(e.currentTarget.value)}
							/>
						</div>
						<FormFeedback error={addForm.error()} />
						<DialogFooter>
							<Button
								type="button"
								variant="outline"
								onClick={() => {
									setShowAdd(false);
									resetForm();
								}}
							>
								Cancel
							</Button>
							<Button
								type="submit"
								disabled={addForm.pending() || !newUrl().trim()}
							>
								{addForm.pending() ? "Adding…" : "Add"}
							</Button>
						</DialogFooter>
					</form>
				</DialogContent>
			</Dialog>
		</div>
	);
}
