import React from "react";
import { ExternalLink, GitFork, Mail, Globe, RefreshCw } from "lucide-react";
import { useCheckAppUpdate } from "../features/app-about/use-check-app-update.js";

const APP_VERSION = import.meta.env.VITE_APP_VERSION ?? "0.1.0";

const LINKS = [
  {
    icon: Mail,
    label: "Email",
    href: "mailto:thien.tranhung@gmail.com",
    display: "thien.tranhung@gmail.com",
  },
  {
    icon: GitFork,
    label: "GitHub",
    href: "https://github.com/thientranhung/astraler-skillbox",
    display: "github.com/thientranhung/astraler-skillbox",
  },
  {
    icon: Globe,
    label: "Blog",
    href: "https://blog.thisistool.com",
    display: "blog.thisistool.com",
  },
] as const;

const APP_UPDATE_ERROR_LABEL: Record<string, string> = {
  network_error: "No internet connection — could not reach GitHub.",
  no_releases: "No releases found on GitHub.",
  http_error: "GitHub returned an unexpected response.",
  parse_error: "Could not parse the release information.",
};

const UPDATE_STATUS_LABEL: Record<string, string> = {
  idle: "",
  checking: "Checking…",
  "up-to-date": "You're up to date ✓",
  available: "New version available!",
  error: "Could not check for updates",
};

export function AboutScreen(): React.JSX.Element {
  const update = useCheckAppUpdate();

  return (
    <div className="p-6 space-y-8 max-w-lg">
      {/* Header */}
      <div>
        <h2 className="text-base font-semibold text-zinc-900">Skillbox</h2>
        <p className="mt-1 text-sm text-zinc-500">Version {APP_VERSION}</p>
      </div>

      {/* Author */}
      <div>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-zinc-400 mb-3">
          Author
        </h3>
        <div className="divide-y divide-zinc-100 rounded border border-zinc-200">
          {LINKS.map(({ icon: Icon, label, href, display }) => (
            <button
              key={label}
              onClick={() => window.open(href, "_blank")}
              className="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-zinc-50"
            >
              <Icon size={14} className="shrink-0 text-zinc-400" />
              <div className="flex flex-1 items-center justify-between gap-2 min-w-0">
                <div>
                  <span className="text-xs font-medium text-zinc-500 mr-2">{label}</span>
                  <span className="text-sm text-zinc-700">{display}</span>
                </div>
                <ExternalLink size={12} className="shrink-0 text-zinc-300" />
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* App update check */}
      <div>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-zinc-400 mb-3">
          App Updates
        </h3>
        <div className="rounded border border-zinc-200 px-4 py-3 space-y-3">
          <div className="flex items-center justify-between gap-4">
            <div className="text-sm text-zinc-700">
              Current version:{" "}
              <span className="font-mono text-xs">{update.currentVersion ?? APP_VERSION}</span>
            </div>
            <button
              onClick={() => update.check()}
              disabled={update.isPending}
              className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50 disabled:opacity-50"
            >
              <RefreshCw size={12} className={update.isPending ? "animate-spin" : ""} />
              Check for Updates
            </button>
          </div>

          {update.status !== "idle" && (
            <div
              className={`text-sm ${
                update.status === "available"
                  ? "text-emerald-700"
                  : update.status === "error"
                    ? "text-amber-700"
                    : "text-zinc-600"
              }`}
            >
              {update.status === "error"
                ? (update.errorCode != null && APP_UPDATE_ERROR_LABEL[update.errorCode]
                    ? APP_UPDATE_ERROR_LABEL[update.errorCode]
                    : UPDATE_STATUS_LABEL["error"])
                : UPDATE_STATUS_LABEL[update.status]}
              {update.status === "available" && update.latestVersion && (
                <span className="ml-1 font-mono text-xs">({update.latestVersion})</span>
              )}
            </div>
          )}

          {update.status === "available" && update.releaseUrl && (
            <button
              onClick={() => window.open(update.releaseUrl!, "_blank")}
              className="flex items-center gap-1.5 text-sm text-blue-600 hover:underline"
            >
              <ExternalLink size={12} />
              View release
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
