import { useQuery } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";
import { useActiveHost } from "../skill-host/use-active-host.js";

export function useSkillsList() {
  const activeHost = useActiveHost();

  return useQuery({
    queryKey: queryKeys.skills.list(activeHost?.hostId ?? 0),
    queryFn: () => methods.listSkills({ hostId: activeHost!.hostId }),
    enabled: activeHost != null,
  });
}
