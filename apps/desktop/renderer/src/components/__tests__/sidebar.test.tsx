// @vitest-environment happy-dom
import { describe, it, expect } from "vitest";
import { NAV_ITEMS } from "../sidebar.js";

describe("NAV_ITEMS", () => {
  it("has Dashboard as the first item", () => {
    expect(NAV_ITEMS[0].to).toBe("/dashboard");
    expect(NAV_ITEMS[0].label).toBe("Dashboard");
  });
});
