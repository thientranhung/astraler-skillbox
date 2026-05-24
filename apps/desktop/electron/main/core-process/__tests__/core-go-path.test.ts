import { describe, it, expect } from "vitest";
import path from "path";
import { resolveCoreGoPath } from "../core-go-path.js";

describe("resolveCoreGoPath", () => {
  it("maps electron-vite out/main/ to repo core-go", () => {
    // electron-vite builds main process to <project>/out/main/
    const outMain = "/Users/dev/astraler-skillbox/apps/desktop/out/main";
    const result = resolveCoreGoPath(outMain);
    expect(result).toBe(path.normalize("/Users/dev/astraler-skillbox/core-go"));
  });

  it("resolves correctly on an arbitrary nested path", () => {
    const outMain = "/home/ci/repo/apps/desktop/out/main";
    expect(resolveCoreGoPath(outMain)).toBe(path.normalize("/home/ci/repo/core-go"));
  });
});
