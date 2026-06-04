import { describe, it, expect } from "vitest";
import { buildDiagnosticsText } from "../diagnostics.js";

const baseOpts = {
  appVersion: "0.1.2",
  electronVersion: "33.0.0",
  chromeVersion: "130.0.0",
  nodeVersion: "22.0.0",
  platform: "darwin",
  arch: "arm64",
  dbPath: "/Users/alice/Library/Application Support/Astraler Skillbox/skillbox.db",
  homeDir: "/Users/alice",
  exportedAt: "2026-06-05T00:00:00.000Z",
  coreLogLines: [],
};

describe("buildDiagnosticsText", () => {
  it("includes app version and platform info", () => {
    const text = buildDiagnosticsText(baseOpts);
    expect(text).toContain("App version: 0.1.2");
    expect(text).toContain("Platform: darwin arm64");
    expect(text).toContain("Electron: 33.0.0");
    expect(text).toContain("Node: 22.0.0");
  });

  it("redacts home dir from db path", () => {
    const text = buildDiagnosticsText(baseOpts);
    expect(text).not.toContain("/Users/alice");
    expect(text).toContain("~/Library/Application Support/Astraler Skillbox/skillbox.db");
  });

  it("redacts home dir from log lines", () => {
    const text = buildDiagnosticsText({
      ...baseOpts,
      coreLogLines: ["time=2026 level=INFO msg=opened path=/Users/alice/.claude/skills"],
    });
    expect(text).not.toContain("/Users/alice/.claude");
    expect(text).toContain("~/.claude/skills");
  });

  it("shows no output captured when log lines are empty", () => {
    const text = buildDiagnosticsText({ ...baseOpts, coreLogLines: [] });
    expect(text).toContain("(no output captured)");
  });

  it("includes exported timestamp", () => {
    const text = buildDiagnosticsText(baseOpts);
    expect(text).toContain("Exported: 2026-06-05T00:00:00.000Z");
  });

  it("includes diagnostics header", () => {
    const text = buildDiagnosticsText(baseOpts);
    expect(text).toContain("=== Astraler Skillbox Diagnostics ===");
    expect(text).toContain("=== Core Log Tail (last 100 lines) ===");
  });

  it("handles empty homeDir without crashing", () => {
    const text = buildDiagnosticsText({ ...baseOpts, homeDir: "" });
    expect(text).toContain("/Users/alice/Library");
  });

  it("joins multiple log lines", () => {
    const text = buildDiagnosticsText({
      ...baseOpts,
      coreLogLines: ["line one", "line two", "line three"],
    });
    expect(text).toContain("line one\nline two\nline three");
  });
});
