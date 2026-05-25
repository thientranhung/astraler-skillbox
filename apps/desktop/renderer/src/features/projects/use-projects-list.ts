import { useQuery } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";

export function useProjectsList() {
  return useQuery({
    queryKey: queryKeys.projects.list(),
    queryFn: () => methods.listProjects(),
  });
}
