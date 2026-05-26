// @vitest-environment happy-dom
import { describe, it, expect } from "vitest";
import { NAV_ITEMS } from "../sidebar.js";

describe("NAV_ITEMS", () => {
  it("has Dashboard as the first item", () => {
    expect(NAV_ITEMS[0].to).toBe("/dashboard");
    expect(NAV_ITEMS[0].label).toBe("Dashboard");
  });

  it("has Global Skills between Skills and Projects", () => {
    const labels = NAV_ITEMS.map((item) => item.label);
    const skillsIdx = labels.indexOf("Skills");
    const globalIdx = labels.indexOf("Global Skills");
    const projectsIdx = labels.indexOf("Projects");
    expect(globalIdx).toBeGreaterThan(skillsIdx);
    expect(globalIdx).toBeLessThan(projectsIdx);
    expect(NAV_ITEMS[globalIdx].to).toBe("/global");
  });

  it("has Plugins after Projects and before Settings", () => {
    const labels = NAV_ITEMS.map((item) => item.label);
    const projectsIdx = labels.indexOf("Projects");
    const pluginsIdx = labels.indexOf("Plugins");
    const settingsIdx = labels.indexOf("Settings");
    expect(pluginsIdx).toBeGreaterThan(projectsIdx);
    expect(pluginsIdx).toBeLessThan(settingsIdx);
    expect(NAV_ITEMS[pluginsIdx].to).toBe("/plugins");
  });
});
