import { useQuery } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";

export function useSkillDetail(skillId: number | null) {
  return useQuery({
    queryKey: queryKeys.skills.detail(skillId ?? 0),
    queryFn: () => methods.getSkill({ skillId: skillId! }),
    enabled: skillId != null,
  });
}
