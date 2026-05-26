// @vitest-environment happy-dom
import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import React from "react";
import { ProviderIcon } from "../provider-icon.js";

describe("ProviderIcon", () => {
  it("renders a brand SVG asset for known brand providers", () => {
    const { container } = render(<ProviderIcon providerKey="claude" />);

    expect(container.querySelector("img")).not.toBeNull();
    expect(container.querySelector("svg")).toBeNull();
  });

  it("renders the generic agent fallback for generic_agents", () => {
    const { container } = render(<ProviderIcon providerKey="generic_agents" />);

    expect(container.querySelector("img")).toBeNull();
    expect(container.querySelector("svg")).not.toBeNull();
  });
});
