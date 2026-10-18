import { Skeleton, SkeletonList } from "@/components/ui/skeleton";

export function PageSkeleton() {
	return (
		<div class="px-7 py-6">
			<Skeleton class="mb-2 h-6 w-48" />
			<Skeleton class="mb-6 h-4 w-72" />
			<SkeletonList rows={6} />
		</div>
	);
}
