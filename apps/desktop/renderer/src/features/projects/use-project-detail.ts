import { useQuery } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";

export function useProjectDetail(projectId: number | null) {
  return useQuery({
    queryKey: queryKeys.projects.detail(projectId ?? 0),
    queryFn: () => methods.getProject({ projectId: projectId! }),
    enabled: projectId != null,
  });
}
