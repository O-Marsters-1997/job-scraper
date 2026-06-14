import {
  createMutation,
  createQuery,
  queryOptions,
  useQueryClient,
} from "@tanstack/solid-query";
import {
  addTrackedDoc,
  fetchCVTemplates,
  hideTab,
  removeTrackedDoc,
  showTab,
} from "../api/cvTemplates";

export const cvTemplatesQueryOptions = queryOptions({
  queryKey: ["cv-templates"],
  queryFn: fetchCVTemplates,
});

export function useCVTemplates() {
  return createQuery(() => cvTemplatesQueryOptions);
}

export function useAddTrackedDoc() {
  const queryClient = useQueryClient();
  return createMutation(() => ({
    mutationFn: (url: string) => addTrackedDoc(url),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["cv-templates"] }),
  }));
}

export function useRemoveTrackedDoc() {
  const queryClient = useQueryClient();
  return createMutation(() => ({
    mutationFn: (docId: string) => removeTrackedDoc(docId),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["cv-templates"] }),
  }));
}

export function useHideTab() {
  const queryClient = useQueryClient();
  return createMutation(() => ({
    mutationFn: ({ docId, tabId }: { docId: string; tabId: string }) =>
      hideTab(docId, tabId),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["cv-templates"] }),
  }));
}

export function useShowTab() {
  const queryClient = useQueryClient();
  return createMutation(() => ({
    mutationFn: ({ docId, tabId }: { docId: string; tabId: string }) =>
      showTab(docId, tabId),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["cv-templates"] }),
  }));
}
