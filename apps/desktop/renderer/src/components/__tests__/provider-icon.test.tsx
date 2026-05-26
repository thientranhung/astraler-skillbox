// @vitest-environment happy-dom
import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import React from "react";
import { ProviderIcon } from "../provider-icon.js";

describe("ProviderIcon", () => {
  it("renders a brand SVG asset for known brand providers", () => {
    const { container } = render(<ProviderIcon providerKey="claude" />);

    expect(container.querySelector("img")).toBeNull();
    expect(container.querySelector("svg")).not.toBeNull();
    expect(container.innerHTML).toContain("Claude");
  });

  it("renders the generic agent fallback for generic_agents", () => {
    const { container } = render(<ProviderIcon providerKey="generic_agents" />);

    expect(container.querySelector("img")).toBeNull();
    expect(container.querySelector("svg")).not.toBeNull();
  });

  it("uses iconKey when provided instead of providerKey for lookup", () => {
    // iconKey="claude" on an unknown providerKey should resolve to Claude SVG
    const { container } = render(<ProviderIcon providerKey="unknown_provider_xyz" iconKey="claude" />);

    expect(container.innerHTML).toContain("Claude");
  });

  it("falls back to Bot icon when neither providerKey nor iconKey maps to a brand icon", () => {
    const { container } = render(<ProviderIcon providerKey="unknown_key" iconKey="also_unknown" />);

    // Bot fallback renders a lucide SVG, not a branded one
    const svg = container.querySelector("svg");
    expect(svg).not.toBeNull();
    // no Claude, codex, gemini, antigravity content
    expect(container.innerHTML).not.toContain("Claude");
  });

  it("falls back to providerKey lookup when iconKey is null", () => {
    const { container } = render(<ProviderIcon providerKey="claude" iconKey={null} />);

    expect(container.innerHTML).toContain("Claude");
  });

  it("renders Bot fallback for opencode (no icon asset yet)", () => {
    const { container } = render(<ProviderIcon providerKey="opencode" iconKey="opencode" />);

    // No brand SVG for opencode — should show Bot fallback
    const svg = container.querySelector("svg");
    expect(svg).not.toBeNull();
    expect(container.innerHTML).not.toContain("Claude");
  });
});
