import { Show } from "solid-js";
import { FactRow } from "@/components/FactRow";
import { SourceBadge } from "@/components/SourceBadge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectItemLabel,
	SelectTrigger,
} from "@/components/ui/select";
import { formatDate } from "@/lib/datetime";
import { useSetCompanyTracking } from "../../../hooks/useCompanies";
import type { Company } from "../../../types/company";

const CHECK_INTERVAL_OPTIONS = [
	{ minutes: 60, label: "Hourly" },
	{ minutes: 180, label: "Every 3 hours" },
	{ minutes: 360, label: "Every 6 hours" },
	{ minutes: 720, label: "Every 12 hours" },
	{ minutes: 1440, label: "Daily" },
];

export function CompanyDetailsCard(props: { company: Company }) {
	const trackMutation = useSetCompanyTracking();

	return (
		<>
			<Card>
				<CardHeader class="pb-2">
					<CardTitle>Details</CardTitle>
				</CardHeader>
				<CardContent class="gap-2.5">
					<FactRow label="ATS">
						<Show when={props.company.ATSSource} fallback="—">
							{(source) => <SourceBadge source={source()} />}
						</Show>
					</FactRow>
					<Show when={props.company.ATSToken}>
						{(token) => (
							<FactRow label="Board token">
								<span class="font-mono text-xs">{token()}</span>
							</FactRow>
						)}
					</Show>
					<FactRow label="First seen">
						{formatDate(props.company.FirstSeenAt)}
					</FactRow>
					<FactRow label="Jobs">{props.company.JobCount}</FactRow>
					<Show when={props.company.Tracked && props.company.LastCheckedAt}>
						{(lastChecked) => (
							<FactRow label="Last checked">
								{formatDate(lastChecked())}
							</FactRow>
						)}
					</Show>
				</CardContent>
			</Card>

			<Show when={props.company.Tracked}>
				<Card>
					<CardHeader class="pb-2">
						<CardTitle>Check frequency</CardTitle>
					</CardHeader>
					<CardContent>
						<Select
							options={CHECK_INTERVAL_OPTIONS}
							optionValue="minutes"
							optionTextValue="label"
							value={
								CHECK_INTERVAL_OPTIONS.find(
									(o) => o.minutes === props.company.CheckIntervalMinutes,
								) ?? null
							}
							onChange={(opt) => {
								if (!opt) return;
								trackMutation.mutate({
									id: props.company.ID,
									enabled: props.company.Tracked,
									checkIntervalMinutes: opt.minutes,
								});
							}}
							itemComponent={(itemProps) => (
								<SelectItem item={itemProps.item}>
									<SelectItemLabel>
										{itemProps.item.rawValue.label}
									</SelectItemLabel>
								</SelectItem>
							)}
						>
							<SelectTrigger>
								<Select.Value<(typeof CHECK_INTERVAL_OPTIONS)[number]>>
									{(state) => state.selectedOption()?.label ?? "Select"}
								</Select.Value>
							</SelectTrigger>
							<SelectContent />
						</Select>
					</CardContent>
				</Card>
			</Show>
		</>
	);
}
