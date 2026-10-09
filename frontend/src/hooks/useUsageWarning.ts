import { type Accessor, createContext, createMemo, useContext } from "solid-js";
import { quotaLevel, worstLevel } from "@/lib/usageLevel";
import type { UsageLevel } from "@/types/quota";
import { useAiUsage } from "./useAiUsage";
import { useIsAdmin } from "./useIsAdmin";
import { useProxyUsage } from "./useProxyUsage";

export interface UsageWarning {
	overall: Accessor<UsageLevel>;
	forSection: (to: string) => UsageLevel;
}

export function createUsageWarning(): UsageWarning {
	const isAdmin = useIsAdmin();
	const ai = useAiUsage();
	const proxy = useProxyUsage();
	const aiLevel = createMemo(() => quotaLevel(ai.data));
	const proxyLevel = createMemo<UsageLevel>(() =>
		isAdmin() ? worstLevel(proxy.data?.providers.map(quotaLevel) ?? []) : "ok",
	);
	const overall = createMemo(() => worstLevel([aiLevel(), proxyLevel()]));
	return {
		overall,
		forSection: (to) =>
			to === "/settings/ai"
				? aiLevel()
				: to === "/settings/usage"
					? proxyLevel()
					: "ok",
	};
}

const NO_WARNING: UsageWarning = {
	overall: () => "ok",
	forSection: () => "ok",
};

export const UsageWarningContext = createContext<UsageWarning>(NO_WARNING);

export function useUsageWarning(): UsageWarning {
	return useContext(UsageWarningContext);
}
