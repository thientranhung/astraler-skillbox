import React from "react";
import { RefreshCw } from "lucide-react";
import { useProviderPluginList } from "../features/provider-plugins/use-provider-plugin-list.js";
import { useScanProviderPluginsGlobal } from "../features/provider-plugins/use-scan-provider-plugins-global.js";
import { ErrorDisplay } from "../components/error-display.js";
import { EmptyState } from "../components/empty-state.js";
import { ProviderIcon } from "../components/provider-icon.js";
import type { PPGlobalView, PPGlobalEntry, PPMarketplace } from "@contracts/index.js";

const LAYER_STATUS_LABEL: Record<string, string> = {
  ok: "ok",
  missing: "not configured",
  unreadable: "unreadable",
  malformed: "malformed",
  too_large: "too large",
  symlink: "symlink (skipped)",
  path_escape: "path escape (skipped)",
};

function layerStatusClass(status: string): string {
  switch (status) {
    case "ok": return "bg-green-100 text-green-700";
    case "missing": return "bg-zinc-100 text-zinc-400";
    case "unreadable":
    case "malformed":
    case "too_large":
    case "symlink":
    case "path_escape": return "bg-yellow-100 text-yellow-700";
    default: return "bg-zinc-100 text-zinc-500";
  }
}

function pluginStatusClass(status: PPGlobalEntry["status"]): string {
  return status === "enabled" ? "bg-green-100 text-green-700" : "bg-zinc-100 text-zinc-500";
}

function MarketplaceRow({ m }: { m: PPMarketplace }): React.JSX.Element {
  return (
    <tr className="border-b border-zinc-100">
      <td className="px-3 py-1.5 text-xs font-medium text-zinc-700">{m.marketplaceName}</td>
      <td className="px-3 py-1.5 text-xs text-zinc-500">{m.sourceType}</td>
      <td className="px-3 py-1.5 font-mono text-xs text-zinc-400">{m.sourceSummary}</td>
    </tr>
  );
}

function GlobalPluginView({ global: g }: { global: PPGlobalView }): React.JSX.Element {
  const neverScanned = g.userLayerStatus == null;
  const statusLabel = neverScanned
    ? "never scanned"
    : LAYER_STATUS_LABEL[g.userLayerStatus!] ?? g.userLayerStatus!;
  const statusClass = neverScanned ? "bg-zinc-100 text-zinc-400" : layerStatusClass(g.userLayerStatus!);

  return (
    <div className="flex flex-col gap-4">
      {/* Provider identity + user layer */}
      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <span className="inline-flex items-center gap-1.5 text-sm font-medium text-zinc-900">
            <ProviderIcon providerKey={g.providerKey} />
            {g.providerKey === "claude" ? "Claude" : g.providerKey}
          </span>
          <span className={`rounded px-1.5 py-0.5 text-xs font-medium ${statusClass}`}>
            {statusLabel}
          </span>
          {g.lastScannedAt != null && (
            <span className="text-xs text-zinc-400">
              Scanned {new Date(g.lastScannedAt).toLocaleString()}
            </span>
          )}
        </div>
        <p className="font-mono text-xs text-zinc-400">{g.userLayerPath}</p>
        {g.managedOutOfScope && (
          <p className="text-xs text-zinc-400">
            Some settings in this file are managed outside Skillbox.
          </p>
        )}
      </div>

      {/* Scan notes — non-alarming for ok/missing, plain for bad statuses */}
      {g.scanWarnings.length > 0 && g.userLayerStatus !== "missing" && (
        <div className="flex flex-col gap-0.5 rounded border border-zinc-200 bg-zinc-50 px-3 py-2">
          <span className="mb-1 text-xs font-medium text-zinc-500">Scan notes</span>
          {g.scanWarnings.map((w, i) => (
            <p key={i} className="text-xs text-zinc-500">{w}</p>
          ))}
        </div>
      )}

      {/* Plugins table */}
      {g.plugins.length > 0 && (
        <div>
          <h4 className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-zinc-500">
            Plugins
          </h4>
          <div className="overflow-x-auto rounded border border-zinc-200">
            <table className="w-full text-left">
              <thead className="border-b border-zinc-200 bg-zinc-50">
                <tr>
                  <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Plugin</th>
                  <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Marketplace</th>
                  <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Status</th>
                </tr>
              </thead>
              <tbody>
                {g.plugins.map((p, i) => (
                  <tr key={i} className="border-b border-zinc-100 hover:bg-zinc-50">
                    <td className="px-3 py-1.5 text-xs font-medium text-zinc-900">{p.pluginName}</td>
                    <td className="px-3 py-1.5 text-xs text-zinc-500">{p.marketplaceName || "—"}</td>
                    <td className="px-3 py-1.5 text-xs">
                      <span className={`rounded px-1.5 py-0.5 font-medium ${pluginStatusClass(p.status)}`}>
                        {p.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {g.plugins.length === 0 && g.userLayerStatus === "ok" && (
        <p className="text-xs text-zinc-400">No plugins declared in settings.</p>
      )}

      {/* Marketplaces */}
      {g.marketplaces.length > 0 && (
        <div>
          <h4 className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-zinc-500">
            Marketplaces
          </h4>
          <div className="overflow-x-auto rounded border border-zinc-200">
            <table className="w-full text-left">
              <thead className="border-b border-zinc-200 bg-zinc-50">
                <tr>
                  <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Name</th>
                  <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Type</th>
                  <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Source</th>
                </tr>
              </thead>
              <tbody>
                {g.marketplaces.map((m, i) => (
                  <MarketplaceRow key={i} m={m} />
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

export function PluginsScreen(): React.JSX.Element {
  const { data, isPending, isError, error } = useProviderPluginList();
  const scanMutation = useScanProviderPluginsGlobal();
  const isScanning = scanMutation.operationId != null || scanMutation.isPending;

  return (
    <div className="flex flex-1 flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
        <div>
          <h2 className="text-sm font-semibold text-zinc-900">Provider Plugins</h2>
          <p className="mt-0.5 text-xs text-zinc-400">
            Read-only view of Claude plugin settings. Scan to refresh.
          </p>
        </div>
        <button
          onClick={() => scanMutation.mutate()}
          disabled={isScanning}
          className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
        >
          <RefreshCw size={13} className={isScanning ? "animate-spin" : ""} />
          {isScanning ? "Scanning…" : "Scan Global"}
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {isPending && (
          <div className="flex h-40 items-center justify-center">
            <div className="h-5 w-5 animate-spin rounded-full border-2 border-zinc-300 border-t-zinc-700" />
          </div>
        )}

        {isError && (
          <div className="p-4">
            <ErrorDisplay error={error} />
          </div>
        )}

        {!isPending && !isError && data == null && (
          <EmptyState
            message="No plugin data."
            description="Run Scan Global to read Claude plugin settings."
          />
        )}

        {!isPending && !isError && data != null && (
          <div className="p-4">
            <GlobalPluginView global={data.global} />
          </div>
        )}
      </div>
    </div>
  );
}
