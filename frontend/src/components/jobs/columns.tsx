import type { ColumnDef } from "@tanstack/solid-table";
import { Badge } from "@/components/ui/badge";
import { JobActionsMenu } from "./JobActionsMenu";
import type { Job } from "@/types/job";
import type { JobApplicationSummary } from "@/types/application";

export interface JobTableContext {
  appsForJobs: () => Record<string, JobApplicationSummary> | undefined;
  onTrack: (jobId: string) => void;
  onEdit: (jobId: string) => void;
}

export function createJobColumns(
  ctx: JobTableContext,
): ColumnDef<Job, unknown>[] {
  return [
    {
      accessorKey: "Title",
      header: "Title",
      cell: (info) => (
        <span class="font-medium text-[var(--sea-ink)]">
          {info.getValue() as string}
        </span>
      ),
    },
    {
      accessorKey: "CompanySlug",
      header: "Company",
    },
    {
      accessorKey: "Location",
      header: "Location",
      cell: (info) => (
        <span class="text-[var(--sea-ink-soft)]">
          {info.getValue() as string}
        </span>
      ),
    },
    {
      accessorKey: "Source",
      header: "Source",
      cell: (info) => (
        <Badge variant="secondary" class="uppercase tracking-wide">
          {info.getValue() as string}
        </Badge>
      ),
    },
    {
      accessorKey: "ScrapedAt",
      header: "Date",
      enableGlobalFilter: false,
      cell: (info) => {
        const raw = info.getValue() as string;
        return (
          <span class="text-[var(--sea-ink-soft)]">
            {new Date(raw).toLocaleDateString("en-GB", {
              day: "numeric",
              month: "short",
              year: "numeric",
            })}
          </span>
        );
      },
    },
    {
      id: "actions",
      enableSorting: false,
      enableGlobalFilter: false,
      cell: (info) => (
        <div class="flex justify-end">
          <JobActionsMenu
            job={info.row.original}
            appSummary={ctx.appsForJobs()?.[info.row.original.ID]}
            onTrack={() => ctx.onTrack(info.row.original.ID)}
            onEdit={() => ctx.onEdit(info.row.original.ID)}
          />
        </div>
      ),
    },
  ];
}
