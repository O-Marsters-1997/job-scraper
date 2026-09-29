import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Switch,
	SwitchControl,
	SwitchLabel,
	SwitchThumb,
} from "@/components/ui/switch";

export function CVTemplatesToolbar(props: {
	searchQuery: string;
	onSearch: (q: string) => void;
	showHidden: boolean;
	onShowHidden: (v: boolean) => void;
}) {
	return (
		<div class="mb-3 flex items-center gap-3">
			<Label for="cv-search" class="sr-only">
				Search CVs
			</Label>
			<Input
				id="cv-search"
				type="search"
				placeholder="Search by title or source…"
				value={props.searchQuery}
				onInput={(e) => props.onSearch(e.currentTarget.value)}
				class="max-w-xs"
			/>
			<Switch checked={props.showHidden} onChange={props.onShowHidden}>
				<SwitchLabel class="inline-flex items-center gap-2">
					<SwitchControl>
						<SwitchThumb />
					</SwitchControl>
					Show hidden
				</SwitchLabel>
			</Switch>
		</div>
	);
}
