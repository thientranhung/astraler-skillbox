import React from "react";
import { Bot } from "lucide-react";
import antigravityIcon from "../assets/provider-icons/antigravity-color.svg?raw";
import claudeIcon from "../assets/provider-icons/claude-color.svg?raw";
import codexIcon from "../assets/provider-icons/codex-color.svg?raw";
import geminiIcon from "../assets/provider-icons/gemini-color.svg?raw";

const BRAND_PROVIDER_ICONS: Record<string, string> = {
  antigravity: antigravityIcon,
  antigravity_cli: antigravityIcon,
  claude: claudeIcon,
  claude_code: claudeIcon,
  claudecode: claudeIcon,
  codex: codexIcon,
  gemini: geminiIcon,
  gemini_cli: geminiIcon,
  geminicli: geminiIcon,
};

interface ProviderIconProps {
  providerKey: string;
  className?: string;
}

export function ProviderIcon({
  providerKey,
  className = "",
}: ProviderIconProps): React.JSX.Element {
  const iconSvg = BRAND_PROVIDER_ICONS[providerKey];
  const cls = `inline-flex h-4 w-4 shrink-0 items-center justify-center ${className}`;

  if (iconSvg != null) {
    return (
      <span
        className={`${cls} text-base [&>svg]:h-4 [&>svg]:w-4 [&>svg]:shrink-0`}
        aria-hidden="true"
        dangerouslySetInnerHTML={{ __html: iconSvg }}
      />
    );
  }

  return (
    <span className={`${cls} text-zinc-500`} aria-hidden="true">
      <Bot size={14} strokeWidth={1.8} />
    </span>
  );
}
