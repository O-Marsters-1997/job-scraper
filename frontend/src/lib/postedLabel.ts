import { format } from "date-fns";
import type { Job } from "@/types/job";
import { formatRelative } from "./datetime";

export interface PostedLabel {
	list: string;
	listTitle?: string;
	detail: string;
	detailRepost?: string | undefined;
}

type PostedFields = Pick<
	Job,
	"PostedAt" | "PostedAfter" | "RepostedAt" | "FirstDiscoveredAt" | "ScrapedAt"
>;

const day = (d: Date) => format(d, "d MMM");
const time = (d: Date) => format(d, "HH:mm");

function isDayPrecision(iso: string): boolean {
	return iso.slice(11, 19) === "00:00:00";
}

function windowText(from: Date, to: Date): string {
	const end = day(from) === day(to) ? time(to) : `${day(to)} ${time(to)}`;
	return `${day(from)} ${time(from)} and ${end}`;
}

export function postedLabel(job: PostedFields): PostedLabel {
	const discovered = new Date(job.FirstDiscoveredAt ?? job.ScrapedAt);
	const repost = job.RepostedAt
		? {
				list: ` · Reposted ${formatRelative(job.RepostedAt)}`,
				detail: `Reposted ${day(new Date(job.RepostedAt))}`,
			}
		: undefined;
	const withRepost = (
		label: Omit<PostedLabel, "detailRepost">,
	): PostedLabel => ({
		...label,
		list: label.list + (repost?.list ?? ""),
		detailRepost: repost?.detail,
	});

	if (job.PostedAt) {
		const posted = new Date(job.PostedAt);
		return withRepost({
			list: isDayPrecision(job.PostedAt)
				? day(posted)
				: formatRelative(job.PostedAt),
			detail: `Posted ${day(posted)}`,
		});
	}

	if (job.PostedAfter) {
		const from = new Date(job.PostedAfter);
		const mid = new Date((from.getTime() + discovered.getTime()) / 2);
		const range = windowText(from, discovered);
		return withRepost({
			list: `~${formatRelative(mid.toISOString())}`,
			listTitle: `Posted between ${range}`,
			detail: `Posted between ${range}`,
		});
	}

	return withRepost({
		list: `Before ${day(discovered)}`,
		detail: `Posted on or before ${day(discovered)}`,
	});
}
