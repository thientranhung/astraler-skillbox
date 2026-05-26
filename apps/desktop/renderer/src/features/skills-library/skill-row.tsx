import React from "react";
import { useNavigate } from "@tanstack/react-router";
import type { SkillListSkill } from "@contracts/index.js";
import { SkillStatusBadge } from "./skill-status-badge.js";

interface SkillRowProps {
  skill: SkillListSkill;
}

export function SkillRow({ skill }: SkillRowProps): React.JSX.Element {
  const navigate = useNavigate();

  function handleClick(): void {
    void navigate({ to: "/skills/$skillId", params: { skillId: String(skill.id) } });
  }

  return (
    <tr
      className="cursor-pointer border-b border-zinc-100 hover:bg-zinc-50"
      onClick={handleClick}
    >
      <td className="px-3 py-2 text-sm font-medium text-zinc-900">{skill.name}</td>
      <td className="px-3 py-2">
        <SkillStatusBadge status={skill.status} />
      </td>
      <td className="px-3 py-2 text-sm text-zinc-600">{skill.projectsUsingCount}</td>
      <td className="px-3 py-2 font-mono text-xs text-zinc-500">{skill.relativePath}</td>
      <td className="px-3 py-2 text-sm text-zinc-400">{skill.sourceLabel ?? "—"}</td>
    </tr>
  );
}
