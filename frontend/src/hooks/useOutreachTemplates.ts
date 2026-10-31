import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import {
	createOutreachTemplate,
	deleteOutreachTemplate,
	fetchOutreachTemplates,
	makeOutreachTemplateDefault,
	updateOutreachTemplate,
} from "../api/outreachTemplates";
import type { OutreachTemplateInput } from "../types/outreach";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

const outreachTemplatesQueryOptions = queryOptions({
	queryKey: keys.outreachTemplates,
	queryFn: fetchOutreachTemplates,
});

export function useOutreachTemplates() {
	return createQuery(() => outreachTemplatesQueryOptions);
}

export function useCreateOutreachTemplate() {
	return useInvalidatingMutation(createOutreachTemplate, [
		keys.outreachTemplates,
	]);
}

export function useUpdateOutreachTemplate() {
	return useInvalidatingMutation(
		(v: { id: string } & Pick<OutreachTemplateInput, "name" | "body">) => {
			const { id, ...input } = v;
			return updateOutreachTemplate(id, input);
		},
		[keys.outreachTemplates],
	);
}

export function useMakeOutreachTemplateDefault() {
	return useInvalidatingMutation(makeOutreachTemplateDefault, [
		keys.outreachTemplates,
	]);
}

export function useDeleteOutreachTemplate() {
	return useInvalidatingMutation(deleteOutreachTemplate, [
		keys.outreachTemplates,
	]);
}
