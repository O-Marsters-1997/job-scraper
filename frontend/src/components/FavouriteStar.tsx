import { Icon } from "@/components/Icon";
import { cn } from "@/lib/utils";

interface FavouriteStarProps {
	favourite: boolean;
	companyName: string;
	onToggle: () => void;
	disabled?: boolean;
}

export function FavouriteStar(props: FavouriteStarProps) {
	return (
		<button
			type="button"
			aria-pressed={props.favourite}
			aria-label={`Favourite ${props.companyName}`}
			disabled={props.disabled}
			onClick={() => props.onToggle()}
			class={cn(
				"flex size-6 shrink-0 items-center justify-center rounded-md transition-colors hover:bg-accent-subtle disabled:opacity-50",
				props.favourite ? "text-primary" : "text-faint hover:text-foreground",
			)}
		>
			<Icon
				name="star"
				size={14}
				class={cn(props.favourite && "fill-current")}
			/>
		</button>
	);
}
