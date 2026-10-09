import { SETTINGS_SECTIONS } from "@/lib/settingsSections";
import { useIsAdmin } from "./useIsAdmin";

export function useSettingsGroups() {
	const isAdmin = useIsAdmin();
	return () =>
		SETTINGS_SECTIONS.filter((group) => !group.adminOnly || isAdmin());
}
