import { useQuery } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";
import { useActiveHost } from "../skill-host/use-active-host.js";
import type { SkillListSkill } from "@contracts/index.js";

export interface ActiveHostSkillsResult {
  skills: SkillListSkill[];
  reason: string | null;
  isLoading: boolean;
  error: unknown;
}

export function useActiveHostSkills(): ActiveHostSkillsResult {
  const activeHost = useActiveHost();

  const query = useQuery({
    queryKey: queryKeys.skills.list(activeHost?.hostId ?? 0),
    queryFn: () => methods.listSkills({ hostId: activeHost!.hostId }),
    enabled: activeHost != null,
  });

  if (activeHost == null) {
    return { skills: [], reason: "No active Skill Host configured", isLoading: false, error: null };
  }

  const available = (query.data?.skills ?? []).filter((s) => s.status === "available");
  return { skills: available, reason: null, isLoading: query.isLoading, error: query.error };
}
